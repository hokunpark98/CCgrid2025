import os
import yaml
import requests
from yaml.representer import SafeRepresenter
import urllib.parse as urlparse
from urllib.parse import parse_qs

# Custom representer to handle multiline strings properly
class LiteralUnicode(str):
    pass

def literal_unicode_representer(dumper, data):
    return dumper.represent_scalar('tag:yaml.org,2002:str', data, style='|')

yaml.add_representer(LiteralUnicode, literal_unicode_representer)

def fetch_json_data(url):
    """Fetch JSON data from the given URL."""
    response = requests.get(url)
    response.raise_for_status()  # Raise an error for bad status codes
    return response.json()

def create_service_entry_yaml(replica_data, component_folder, namespace):
    replica_name = replica_data['Replica']
    ip_address = replica_data['IP']
    port = replica_data['Port']

    # Define ServiceEntry structure
    service_entry = {
        'apiVersion': 'networking.istio.io/v1alpha3',
        'kind': 'ServiceEntry',
        'metadata': {
            'name': f"{replica_name}-{port}",
            'namespace': namespace
        },
        'spec': {
            'hosts': [replica_name],
            'addresses': [ip_address],
            'ports': [
                {
                    'number': port,
                    'name': 'http',
                    'protocol': 'HTTP'
                }
            ],
            'resolution': 'STATIC',
            'location': 'MESH_INTERNAL',
            'endpoints': [
                {
                    'address': ip_address,
                    'ports': {
                        'http': port
                    }
                }
            ]
        }
    }

    # Write YAML file
    file_name = os.path.join(component_folder, f"{replica_name}-service-entry.yaml")
    with open(file_name, 'w') as yaml_file:
        yaml.dump(service_entry, yaml_file, sort_keys=False, default_flow_style=False)

def main():
    # URL 설정
    url = "http://localhost:21002/traffic-sched?value=15&namespace=pair&algorithm=hk"

    # URL에서 namespace를 추출
    parsed_url = urlparse.urlparse(url)
    query_params = parse_qs(parsed_url.query)
    namespace = query_params.get("namespace", ["default"])[0]

    # JSON 데이터 가져오기
    json_data = fetch_json_data(url)
    metrics_data = json_data.get('metrics_data', {})

    # Base folder 설정
    base_folder = os.path.join('envoyYaml', namespace, 'envoyentry')
    os.makedirs(base_folder, exist_ok=True)

    # 컴포넌트별 폴더 생성 및 YAML 파일 작성
    for component in metrics_data.get('Components', []):
        component_name = component['Component']
        
        # 각 컴포넌트에 대한 폴더 생성
        component_folder = os.path.join(base_folder, component_name)
        os.makedirs(component_folder, exist_ok=True)

        # 각 레플리카에 대해 ServiceEntry YAML 파일 생성
        for replica in component['Replicas']:
            create_service_entry_yaml(replica, component_folder, namespace)

if __name__ == "__main__":
    main()
