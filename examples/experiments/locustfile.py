import time
import csv
import os
from locust import HttpUser, task, between, events

# Configuration
CSV_FILE = "experiment_results.csv"

# Hook to initialize the CSV file when the test starts
@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    # If file doesn't exist, write headers
    if not os.path.exists(CSV_FILE):
        with open(CSV_FILE, "w", newline="") as f:
            writer = csv.writer(f)
            writer.writerow(["timestamp", "function", "response_time_ms", "node_arch", "status_code", "policy"])

# Hook to capture every request and log custom data
@events.request.add_listener
def on_request(request_type, name, response_time, response_length, response, exception, context, **kwargs):
    if exception:
        return # Don't log failed connection attempts in the main stats for now

    # Extract the Architecture Header from the response
    node_arch = response.headers.get("Serverledge-Node-Arch", "unknown")

    # Identify the policy from environment variable (set before running locust)
    policy = os.environ.get("LB_POLICY", "unknown")

    with open(CSV_FILE, "a", newline="") as f:
        writer = csv.writer(f)
        writer.writerow([
            time.time(),
            name,
            response_time,
            node_arch,
            response.status_code,
            policy
        ])

class ServerledgeUser(HttpUser):

    wait_time = 1

    @task(1)
    def invoke_primenum(self):
        # The name parameter is used to group stats in Locust UI
        self.client.post("/invoke/primenum", json={"params": {}}, name="primenum")

    @task(1)
    def invoke_linpack(self):
        self.client.post("/invoke/linpack", json={"params": {}}, name="linpack")