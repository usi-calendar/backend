import pymongo
import requests
import string
import random
import os
from dotenv import load_dotenv
import json

load_dotenv()

CLIENT = pymongo.MongoClient(os.getenv("MONGO_CONNECTION_STRING"))
DB = CLIENT[os.getenv("MONGO_DB_NAME")]
COL = DB["short_links"]

# URL = "https://api.usicalendar.me/"
URL = "http://localhost:8080/"


def test_random_existing_should_not_add_entry():
    print("[INFO] Make sure the no external connections are allowed during testing")
    if check_for_duplicates() is not None:
        print("[WARNING] The database contains duplicates (more than one shortened link for the same combination of url and courses")
    doc = COL.aggregate([{ "$sample": { "size": 1 } }]).next()
    count1 = COL.count_documents({})
    courses = "~".join(doc["courses"])
    u = doc["url"]
    r = requests.get(f"{URL}shorten?courses={courses}&url={u}")
    if COL.count_documents({}) > count1 and check_for_duplicates() is not None:
        print("[ERROR] The number of elements increased during the test")
        return -1
    print("[LOG] Test passed")

def test_non_existing_shortened():
    r = requests.get(f"{URL}s/{''.join(random.choices(string.ascii_letters + string.digits, k=14))}")
    if r.status_code // 100 == 4:
        print("[LOG] Test passed")
        return
    print("[ERROR] Status code " + str(r.status_code) + " for non-existing shortened link")


def test_shorten_new():
    if check_for_duplicates() is not None:
        print("[ERROR] Can't run this test with duplicates in the database")
        return -1
    print("[LOG] Deleting one random document")
    doc1 = COL.aggregate([{ "$sample": { "size": 1 } }]).next()
    u = doc1["url"]
    courses = "~".join(doc1["courses"])
    COL.delete_one({"_id":doc1["_id"]})

    print("[LOG] Requesting to shorten deleted document")
    requests.get(f"{URL}shorten?url={u}&courses={courses}")

    doc2 = COL.find_one({"url":doc1["url"],"courses":doc1["courses"]})

    if doc2 is None:
        print("[ERROR] No new document was created, inserting old document back")
        COL.insert_one({"_id":doc1["_id"], "url":doc1["url"],"courses":doc1["courses"], "short_url":doc1["short_url"]})
        return -1

    print("[LOG] Adding deleted document back and removing test document")
    
    COL.delete_one({"_id":doc2["_id"]})

    COL.insert_one({"_id":doc1["_id"], "url":doc1["url"],"courses":doc1["courses"], "short_url":doc1["short_url"]})

    print("[LOG] Test passed")

def check_for_duplicates():
    print("[LOG] Checking for duplicates...")
    all = COL.find()
    for doc in all:
        r = COL.find_one({"url":doc["url"],"courses":doc["courses"], "_id": {"$ne":doc["_id"]}})
        if r is not None:
            # id1 = str(r["_id"])
            print("[ERROR] Found duplicate of "+ str(r["_id"]) + " == " + str(doc["_id"]))
            return -1
    print("[LOG] No duplicates found")
        

def test_info_all_calendars():


    res = requests.get(f"{URL}info")
    assert res.status_code == 400

    res = requests.get(f"{URL}info?url=")
    assert res.status_code == 400
    
    res = requests.get(f"{URL}info?url=http://aaa.com")
    assert res.status_code == 400

    res = requests.get(f"{URL}info?url=https://aaa.com")
    assert res.status_code == 400

    FILE_COURSES = "cal_courses.json"
    
    f = open(os.getcwd() + '/static/' + FILE_COURSES)
    data = json.load(f)
    f.close()
    
    print(f"[LOG] Testing all links in {FILE_COURSES}\n")
    total = len(data["cals"]) - 1

    for i, link in enumerate(data["cals"]):
        print(f"{i}/{total}", end="\r")

        res = requests.get(f"{URL}info?url={link}")
        assert res.status_code == 200

    print("[LOG] Done!")


def test_complete_process():

    print(f"[LOG] Testing complete process randomly")

    res = requests.get(f"{URL}courses")
    assert res.status_code == 200
    data = json.loads(res.text)

    course_url = data["cals"][random.randint(0, len(data["cals"])-1)]

    res = requests.get(f"{URL}info?url={course_url}")
    if res.status_code != 200:
        print(f"[ERROR] test_complete_process: {URL}info?url={course_url}")
        return -1

    res = res.text

    subjects = json.loads(res)["courses"]

    choice = [random.randint(0,len(subjects)-1) for _ in range(random.randint(1,len(subjects)+2))]

    f = ""
    for i, c in enumerate(choice):
        s = subjects[c]
        f += s
        if i != len(choice) -1:
            f += "~"

    res = requests.get(f"{URL}shorten?url={course_url}&courses={f}")

    if res.status_code != 200:
        print(f"[ERROR] test_complete_process: {URL}shorten?url={course_url}&courses={f}")
        print(f, choice, subjects)
        return -1

    short_alphanum = json.loads(res.text)["shortened"].split("/")[-1]

    if short_alphanum == "":
        print(f"[ERROR] test_complete_process: short_alphanum is empty")
        return -1

    # doc = COL.find_one({"short_url":short_alphanum})

    # if doc is None:
    #     print(f"[ERROR] test_complete_process: {short_alphanum} is not in DB")
    #     return -1

    res = requests.get(f"{URL}s/{short_alphanum}").status_code
    if res != 200:
        print(f"[ERROR] test_complete_process: {short_alphanum} is not in DB")
        return -1

    # COL.delete_one({"_id":doc["_id"]})
    

    return True

def test_complete_process_n(times):
    for i in range(times):
        if test_complete_process() == -1:
            return -1


# Tests all possible branches of /shorten, ONLY RUN ON DEV DB
def test_shorten_route():
    print("[LOG] Testing all possible branches of /shorten")
    res = requests.get(f"{URL}shorten")
    assert res.status_code == 400

    res = requests.get(f"{URL}shorten?url=&courses=")
    assert res.status_code == 400

    res = requests.get(f"{URL}shorten?url=http://search.usi.ch/&courses=dsadsa~tttyhhh")
    assert res.status_code == 400

    course_url = "https://search.usi.ch/it/offerte-formative/77/master-in-storia-e-teoria-dellarte-e-dellarchitettura-120-ects/piano-orari/53/1/ics"

    res = requests.get(f"{URL}shorten?url={course_url}&courses=dsadsa~tttyhhh")
    assert res.status_code == 400

    res = requests.get(f"{URL}info?url={course_url}").text
    subjects = json.loads(res)["courses"]

    # No "~" is expected at the end of the course list
    res = requests.get(f"{URL}shorten?url={course_url}&courses={subjects[0]}~")
    assert res.status_code == 400
    
    f = ""
    for i, s in enumerate(subjects):
        f += s
        if i != len(subjects) -1:
            f += "~"
    
    res = requests.get(f"{URL}shorten?url={course_url}&courses={f}")

    assert res.status_code == 200

    short_alphanum = json.loads(res.text)["shortened"].split("/")[-1]

    assert short_alphanum != ""

    doc = COL.find_one({"short_url":short_alphanum})

    assert doc

    COL.delete_one({"_id":doc["_id"]})

    return True

def test_s_route():
    print("[LOG] Testing all possible branches of /s")

    res = requests.get(f"{URL}s/a")
    assert res.status_code == 404

    res = requests.get(f"{URL}s")
    assert res.status_code == 404

    res = requests.get(f"{URL}s/")
    assert res.status_code == 404

    doc = COL.aggregate([{ "$sample": { "size": 1 } }]).next()
    rnd_short_url = doc["short_url"]

    res = requests.get(f"{URL}s/{rnd_short_url}")
    assert res.status_code == 200

    return 1



def main():
    assert test_random_existing_should_not_add_entry() != -1
    assert test_non_existing_shortened() != -1
    assert test_shorten_new() != -1
    # assert test_info_all_calendars() != -1
    assert test_shorten_route() == 1
    assert test_s_route() == 1
    assert test_complete_process_n(100) != -1

if __name__ == "__main__":
    main()
