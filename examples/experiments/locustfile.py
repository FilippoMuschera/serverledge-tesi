import time
import csv
import os
from locust import HttpUser, task, between, events, constant

# Configuration
CSV_FILE = "experiment_results.csv"

# Hook to initialize the CSV file when the test starts
@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    # If file doesn't exist, write headers
    if not os.path.exists(CSV_FILE):
        with open(CSV_FILE, "w", newline="") as f:
            writer = csv.writer(f)
            writer.writerow(["timestamp", "function", "response_time_ms", "node_arch", "status_code", "policy", "locust_response_time"])

# Hook to capture every request and log custom data
@events.request.add_listener
def on_request(request_type, name, response_time, response_length, response, exception, context, **kwargs):
    if exception:
        print(f"Request failed: {exception}")
        return

    # Extract the Architecture Header from the response
    node_arch = response.headers.get("Serverledge-Node-Arch", "unknown")

    # Identify the policy from environment variable (set before running locust)
    policy = os.environ.get("LB_POLICY", "unknown")

    serverledge_response_time = "unknown"
    try:
        # Attempt to parse the JSON response to get the server-side response time
        data = response.json()
        if "ResponseTime" in data:
            serverledge_response_time = data["ResponseTime"]
    except Exception:
        pass

    with open(CSV_FILE, "a", newline="") as f:
        writer = csv.writer(f)
        writer.writerow([
            time.time(),
            name,
            serverledge_response_time,
            node_arch,
            response.status_code,
            policy,
            response_time
        ])

class ServerledgeUser(HttpUser):

    wait_time = constant(0.2)

    @task(1)
    def invoke_primenum(self):
        # The name parameter is used to group stats in Locust UI
        self.client.post("/invoke/primenum", json={"params": {}}, name="primenum")

    @task(1)
    def invoke_linpack(self):
        self.client.post("/invoke/linpack", json={"params": {}}, name="linpack")