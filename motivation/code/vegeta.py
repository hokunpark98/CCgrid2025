#!/usr/bin/env python3

import os
import subprocess
import sys
from datetime import datetime
import requests
import json
import time

PROMETHEUS_URL = 'http://10.104.130.33:8080/api/v1/query'

if '-h' in sys.argv or '--help' in sys.argv:
    print('usage:', file=sys.stderr)
    print('echo GET http://localhost:8080/ | %s' % sys.argv[0], file=sys.stderr)
    sys.exit(1)

# Read the target URL from stdin
target = sys.stdin.read().strip()

# Print current time
now = datetime.now()
print("현재 시간: ", now)

# Set the result folder path
result_folder = '/home/dnc/master/CCgrid2024/motivation/results/LEAST_CONN/50/3-5'

# Create the folder if it doesn't exist
os.makedirs(result_folder, exist_ok=True)

# Set duration (in minutes) and rate (requests per minute)
RATE_PER_MINUTES = [100, 200, 300, 400, 500]    
DURATION = 300
NAMESPACE = "pair"

def query_prometheus(promql_query):
    """Send a Prometheus query and return the result as JSON."""
    response = requests.get(PROMETHEUS_URL, params={'query': promql_query})
    print('response', response)
    if response.status_code == 200:
        return response.json()
    else:
        print(f"Error querying Prometheus: {response.status_code}", file=sys.stderr)
        return None

for rate_per_minute in RATE_PER_MINUTES:
    print('RATE_PER_MINUTES', rate_per_minute)
    
    # Convert rate to requests per second for Vegeta
    # Set the output filename for Vegeta results
    vegeta_filename = f'{result_folder}/{rate_per_minute}_results.json'

    # Construct the Vegeta command
    cmd = (
        f'vegeta2 attack -timeout 0s -duration {DURATION}s -rate {rate_per_minute}/1m | '
        f'vegeta2 report -type=json >> {vegeta_filename}'
    )

    # Execute the Vegeta command
    subprocess.run(cmd, shell=True, input=target, encoding='utf-8')
    print('Processing... : 100%')

    # Print the end time
    print("끝난 시간: ", datetime.now())

    # Run Prometheus queries and save results
    promql_queries = {
        'cpu_usage': 'sum(rate(container_cpu_usage_seconds_total{namespace="pair", container="trigono"}[5m])) by (pod) * 100 / 0.3',
        #초당 몇개의 요청이 들어왔는지
        'requests_received': 'sum(rate(istio_requests_total{reporter="destination", destination_workload_namespace="pair", destination_app="trigono"}[5m])) by (destination_workload)'
    }

    # Initialize combined_results for each rate_per_minute
    combined_results = {}

    # Run Prometheus queries only once and save results
    for query_name, promql_query in promql_queries.items():
        prometheus_data = query_prometheus(promql_query)
        if prometheus_data:
            # Extract the second value of the 'value' array
            for result in prometheus_data['data']['result']:
                metric = result['metric']
                pod_name = metric.get('pod', metric.get('destination_workload', 'unknown'))
                value = float(result['value'][1])  # The second value in the 'value' array (actual value)
                
                if pod_name not in combined_results:
                    combined_results[pod_name] = {}
                
                combined_results[pod_name][query_name] = value

    # Save the results for the current rate_per_minute to a file
    final_filename = f'{result_folder}/Util_and_Responsed_{rate_per_minute}.json'
    with open(final_filename, 'w') as f:
        json.dump(combined_results, f, indent=4)
    
    print(f'Results for {rate_per_minute} saved to {final_filename}')

    time.sleep(120) #이전 결과가 영향 주지 않도록