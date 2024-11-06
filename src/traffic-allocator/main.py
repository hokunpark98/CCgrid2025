import os
import urllib.parse as urlparse
from urllib.parse import parse_qs
from fetch import fetch_json_data
from service_entry import create_service_entry_yaml, write_service_entries
from envoy_filter import generate_yaml_files

def main():
    url = "http://localhost:21002/traffic-sched?value=15&namespace=pair&algorithm=hk"
    parsed_url = urlparse.urlparse(url)
    query_params = parse_qs(parsed_url.query)
    namespace = query_params.get("namespace", ["default"])[0]
    
    json_data = fetch_json_data(url)
    
    metrics_data = json_data.get('metrics_data', {})
    final_results = json_data.get('final_results', [])

    # Base folder setup
    base_folder = os.path.join('envoyYaml', namespace, 'envoyentry')
    os.makedirs(base_folder, exist_ok=True)

    # Generate ServiceEntries
    for component in metrics_data.get('Components', []):
        component_name = component['Component']
        service_entries = [create_service_entry_yaml(replica, namespace) for replica in component['Replicas']]
        write_service_entries(service_entries, base_folder, component_name)

    # Generate EnvoyFilter files
    generate_yaml_files(final_results, namespace)

if __name__ == "__main__":
    main()
