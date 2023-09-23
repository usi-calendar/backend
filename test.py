import pymongo
import requests
import string
import random
import os
from dotenv import load_dotenv
import json
from tqdm import tqdm
from ics import Calendar

load_dotenv()

CLIENT = pymongo.MongoClient(os.getenv("MONGO_CONNECTION_STRING"))
DB = CLIENT[os.getenv("MONGO_DB_NAME")]
COL = DB["short_links"]
COMPLEX_COL = DB["complex_short_links"]

URL = os.getenv("TEST_URL")

def test_random_existing_should_not_add_entry():
    print("[INFO] Make sure the no external connections are allowed during testing")
    if check_for_duplicates() is not None:
        print("[WARNING] The database contains duplicates (more than one shortened link for the same combination of url and courses")
    doc = COL.aggregate([{ "$sample": { "size": 1 } }]).next()
    count1 = COL.count_documents({})
    courses = "~".join(doc["subjects"])
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
    courses = "~".join(doc1["subjects"])
    COL.delete_one({"_id":doc1["_id"]})

    print("[LOG] Requesting to shorten deleted document")
    res = requests.get(f"{URL}shorten?url={u}&subjects={courses}")

    assert res.ok

    doc2 = COL.find_one({"url":doc1["url"],"subjects":doc1["subjects"]})

    if doc2 is None:
        print("[ERROR] No new document was created, inserting old document back")
        COL.insert_one({"_id":doc1["_id"], "url":doc1["url"],"subjects":doc1["subjects"], "short_url":doc1["short_url"]})
        return -1

    print("[LOG] Adding deleted document back and removing test document")
    
    COL.delete_one({"_id":doc2["_id"]})

    COL.insert_one({"_id":doc1["_id"], "url":doc1["url"],"subjects":doc1["subjects"], "short_url":doc1["short_url"]})

    print("[LOG] Test passed")

def check_for_duplicates():
    print("[LOG] Checking for duplicates...")
    all = COL.find()
    for doc in all:
        r = COL.find_one({"url":doc["url"],"subjects":doc["subjects"], "_id": {"$ne":doc["_id"]}})
        if r is not None:
            # id1 = str(r["_id"])
            print("[ERROR] Found duplicate of "+ str(r["_id"]) + " == " + str(doc["_id"]))
            return -1
    print("[LOG] No duplicates found")
        

def test_info_all_calendars():


    res = requests.get(f"{URL}urlinfo")
    assert res.status_code == 400

    res = requests.get(f"{URL}urlinfo?url=")
    assert res.status_code == 400
    
    res = requests.get(f"{URL}urlinfo?url=http://aaa.com")
    assert res.status_code == 400

    res = requests.get(f"{URL}urlinfo?url=https://aaa.com")
    assert res.status_code == 400

    # FILE_COURSES = "cal_courses.json"
    
    res = requests.get(f"{URL}courses")
    assert res.status_code == 200
    data = json.loads(res.text)

    assert data["cals"] != None
    
    print(f"[LOG] Testing all {len(data['cals'])} links\n")
    total = len(data["cals"]) - 1

    for link in tqdm(data["cals"]):
        # print(f"{i}/{total}", end="\r")

        res = requests.get(f"{URL}urlinfo?url={link}")
        assert res.status_code == 200

    print("[LOG] Done!")


def test_complete_process():

    # print(f"[LOG] Testing complete process randomly")
    short_alphanum = None
    course_url = None
    try: 
        res = requests.get(f"{URL}courses")
        assert res.ok
        data = json.loads(res.text)

        course_url = random.choice(data["cals"])

        res = requests.get(f"{URL}urlinfo?url={course_url}")
        if res.status_code != 200:
            print(f"[ERROR] test_complete_process: {URL}urlinfo?url={course_url}")
            return -1

        res = res.text

        subjects = json.loads(res)["courses"]

        choice = [random.randint(0,len(subjects)-1) for _ in range(random.randint(1,len(subjects)+2))]

        f = ""
        for i, c in enumerate(choice):
            s = subjects[c][0]
            f += s
            if i != len(choice) -1:
                f += "~"

        expected_status = 200
        if len(choice) > len(set(choice)):
            expected_status = 400

        res = requests.get(f"{URL}shorten?url={course_url}&subjects={f}")

        if res.status_code != expected_status:
            print(f"[ERROR] test_complete_process: {URL}shorten?url={course_url}&subjects={f}")
            print(f, choice, subjects)
            return -1
        
        if expected_status == 400:
            return

        short_alphanum = json.loads(res.text)["shortened"].split("/")[-1]

        if short_alphanum == "":
            print(f"[ERROR] test_complete_process: short_alphanum is empty")
            return -1

        res = requests.get(f"{URL}s/{short_alphanum}").status_code
        if res != 200:
            print(f"[ERROR] test_complete_process: {short_alphanum} is not in DB")
            return -1

        COL.delete_one({"short_url":short_alphanum})
    except Exception as e:
        print(f"[ERROR] Exception with course:{course_url}| alpanum:{short_alphanum}|")
        print(e)
        return -1

    return True

def test_complete_process_n(times):
    for i in tqdm(range(times)):
        if test_complete_process() == -1:
            return -1


# Tests all possible branches of /shorten, ONLY RUN ON DEV DB
def test_shorten_route():
    print("[LOG] Testing all possible branches of /shorten")
    res = requests.get(f"{URL}shorten")
    assert res.status_code == 400

    res = requests.get(f"{URL}shorten?url=&subjects=")
    assert res.status_code == 400

    res = requests.get(f"{URL}shorten?url=http://search.usi.ch/&subjects=dsadsa~tttyhhh")
    assert res.status_code == 400

    res = requests.get(f"{URL}courses")
    assert res.ok
    course_url = random.choice(json.loads(res.text)['cals'])

    res = requests.get(f"{URL}shorten?url={course_url}&subjects=dsadsa~tttyhhh")
    assert res.status_code == 400

    res = requests.get(f"{URL}urlinfo?url={course_url}")
    assert res.ok
    subjects = json.loads(res.text)["courses"]

    # No "~" is expected at the end of the course list
    res = requests.get(f"{URL}shorten?url={course_url}&subjects={subjects[0]}~")
    assert res.status_code == 400
    
    f = ""
    for i, s in enumerate(subjects):
        f += s[0]
        if i != len(subjects) -1:
            f += "~"
    
    res = requests.get(f"{URL}shorten?url={course_url}&subjects={f}")

    assert res.status_code == 200

    short_alphanum = json.loads(res.text)["shortened"].split("/")[-1]

    assert short_alphanum != ""

    doc = COL.find_one({"short_url":short_alphanum})

    assert doc

    COL.delete_one({"_id":doc["_id"]})

    print("[LOG] Test passed")

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

    print("[LOG] Test passed")  

    return 1

# Complex calendar testing


def remove_simple_from_db(id):
    assert COL.delete_one({"short_url":id}).deleted_count == 1

def remove_complex_from_db(id):
    assert COMPLEX_COL.delete_one({"short_url":id}).deleted_count == 1



def test_complex_cal_shorten():

    has_base_cal = random.randint(0,1)

    urlchoice = ""
    base_cal_subjs = []

    hbs = 'false'

    if has_base_cal:

        hbs = 'true'

        res = requests.get(f"{URL}courses")
        assert res.ok
        coursesurls = json.loads(res.text)

        ok = False

        while not ok:
            urlchoice = random.choice(coursesurls['cals'])
            res = requests.get(f"{URL}urlinfo?url={urlchoice}")
            assert res.ok

            info = json.loads(res.text)

            if len(info['courses']) > 1:
                ok = True

        base_cal_subjs = random.sample(info['courses'], random.randint(1, len(info['courses'])))

        base_cal_subjs = list(map(lambda x: x[0], base_cal_subjs))

    res = requests.get(f"{URL}extcourses")
    assert res.ok
    all_subjs = json.loads(res.text)

    all_subjs_selection = list(map(lambda x: x['subjects'],random.sample(all_subjs, random.randint(1,5))))

    all_subjs_selection = list(map(lambda x: random.sample(x, random.randint(1, min(3, len(x)))), all_subjs_selection))

    t = []

    cals = []

    for a in all_subjs_selection:
        for e in a:
            t.append(e)
            res = requests.get(f"https://search.usi.ch/courses/{e}/*/schedules/ics")
            assert res.ok
            cals.append(Calendar(res.text))

    if has_base_cal:
        res = requests.get(f"{URL}shorten?url={urlchoice}&subjects={'~'.join(base_cal_subjs)}")
        assert res.ok
        base_id = json.loads(res.text)['shortened'].split('/')[-1]
        res = requests.get(f"{URL}s/{base_id}")
        assert res.ok
        cals.append(Calendar(res.text))
        

    event_count = 0

    for cal in cals:
        event_count += len(cal.events)

    res = requests.get(f"{URL}cshorten?has_base_calendar={hbs}&url={urlchoice}&subjects={array_to_string(base_cal_subjs)}&extra_subjects={array_to_string(t)}")
    if len(set(t)) != len(t):
        assert res.status_code == 400
        if has_base_cal:
            remove_simple_from_db(base_id)
        return has_base_cal
    else:
        assert res.ok

    id = json.loads(res.text)['shortened'].split('/')[-1]

    res = requests.get(f"{URL}cs/{id}")

    assert res.ok

    full_cal = Calendar(res.text)

    assert len(full_cal.events) == event_count

    if has_base_cal:
        remove_simple_from_db(base_id)
    remove_complex_from_db(id)

    return has_base_cal



def test_complex_cal_shorten_wrapper(count):
    print(f"[LOG] Generating {count} complex calendars")
    with_base_count = 0
    for _ in tqdm(range(count)):
        with_base_count += test_complex_cal_shorten()
    print(f"[LOG] Tested {with_base_count} calendars with base and {count - with_base_count} without a base")
    print("[LOG] Test passed")
    return 1

def array_to_string(ar):
    f = ""
    for i, s in enumerate(ar):
        f += s
        if i != len(ar) -1:
            f += "~"
    return f


def main():
    assert test_random_existing_should_not_add_entry() != -1
    assert test_non_existing_shortened() != -1
    assert test_shorten_new() != -1
    assert test_info_all_calendars() != -1
    assert test_shorten_route() == 1
    assert test_s_route() == 1
    assert test_complete_process_n(100) != -1
    assert test_complex_cal_shorten_wrapper(50) == 1

if __name__ == "__main__":
    main()
