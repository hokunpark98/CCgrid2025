#!/usr/bin/env python3

import os
import subprocess
import sys
from datetime import datetime
import requests
import json
import time

PROMETHEUS_URL = 'http://10.107.204.182:8080/api/v1/query'
MOTIVATION_IP = "10.107.35.212"
NAMESPACE = "pair"
DEFAULT_TARGET_URL = f"GET http://{MOTIVATION_IP}:11000/a?value=1"
RATE_PER_MINUTES = [100, 150, 200, 250, 300]    
DURATION = 60
METRICS_URL = f"http://10.96.9.243:21001/metrics?value={DURATION}&namespace={NAMESPACE}"

now = datetime.now()
print("현재 시간: ", now)

result_folder = '/home/dnc/hokun/CCgrid2025/src/motivation/results'

os.makedirs(result_folder, exist_ok=True)

def fetch_and_save_metrics(metrics_url, output_folder, rate_per_second):
    """
    Fetch metrics from the given URL and save the JSON response to a file.
    """
    try:
        response = requests.get(metrics_url)
        if response.status_code == 200:
            metrics_data = response.json()  # Parse JSON response
            
            # Save JSON to a file
            metrics_filename = f'{output_folder}/replica_{rate_per_second}_results.json'
            with open(metrics_filename, 'w') as metrics_file:
                json.dump(metrics_data, metrics_file, indent=4)
            
            print(f"Metrics saved to {metrics_filename}")
        else:
            print(f"Failed to fetch metrics: {response.status_code}", file=sys.stderr)
    except Exception as e:
        print(f"Error fetching metrics: {e}", file=sys.stderr)


def query_prometheus(promql_query):
    """Send a Prometheus query and return the result as JSON."""
    response = requests.get(PROMETHEUS_URL, params={'query': promql_query})
    print('response', response)
    if response.status_code == 200:
        return response.json()
    else:
        print(f"Error querying Prometheus: {response.status_code}", file=sys.stderr)
        return None

# Use default URL if no input is provided
target = DEFAULT_TARGET_URL

for rate_per_second in RATE_PER_MINUTES:
    print('REQUEST_PER_SECOND', rate_per_second)
    
    # Convert rate to requests per second for Vegeta
    # Set the output filename for Vegeta results
    vegeta_filename = f'{result_folder}/http_{rate_per_second}_results.json'

    # Construct the Vegeta command
    cmd = (
        f'echo "{target}" | '
        f'vegeta attack -timeout 0s -duration {DURATION}s -rate {rate_per_second} | '
        f'vegeta report -type=json >> {vegeta_filename}'
    )

    # Execute the Vegeta command
    subprocess.run(cmd, shell=True)
    print('Processing... : 100%')
    fetch_and_save_metrics(METRICS_URL, result_folder, rate_per_second)

    # Print the end time
    print("끝난 시간: ", datetime.now())
    time.sleep(30)



