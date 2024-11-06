import os
import yaml
from utils import LiteralUnicode

def create_service_entry_yaml(replica_data, namespace):
    """Create a single ServiceEntry YAML structure."""
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
    return service_entry

def write_service_entries(service_entries, base_folder, component_name):
    file_name = os.path.join(base_folder, f"{component_name}-service-entries.yaml")
    with open(file_name, 'w') as yaml_file:
        yaml.dump_all(service_entries, yaml_file, sort_keys=False, default_flow_style=False)
