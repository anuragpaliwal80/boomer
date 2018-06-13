import os
import json
from locust import HttpLocust, TaskSet, task
# from locust.contrib.fasthttp import FastHttpLocust

class ThrashRoot(TaskSet):
  @task
  def make_request(self):
    self.client.get(self.locust.url, verify=False, name=self.locust.url, stream=False)

class DummyLocust(HttpLocust):
    task_set = ThrashRoot
    url = "example.com"

# filepath = "/tmp/test-definitions.json"
# test_data = json.load(open(filepath))
#
# suites = os.environ.get('TEST_SUITES').split(" ")
# i = 0
#
# # Dynamically create a Locust per host/path pair.
# # Locusts are weighted equally and only have one task to run so a "user"
# # only makes one request at a time.
# # Another way to achieve the same thing would have been to dynamically
# # create each task on the TaskSet and have a single Locust.
# # This approach seemed easier.
# for suite in suites:
#   for item in test_data[suite]:
#     class_name = "LocustClient{}".format(i)
#     i = i+1
#
#     globals()[class_name] = type(class_name, (HttpLocust,), {
#       "task_set": ThrashRoot,
#       "url": item["host"] + item["path"]
#     })
