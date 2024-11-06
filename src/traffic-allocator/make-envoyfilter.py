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

def generate_yaml_files(final_results, namespace):
    # Create the base folder structure: envoyYaml/<namespace>/envoyfilter
    base_folder = os.path.join('envoyYaml', namespace, 'envoyfilter')
    os.makedirs(base_folder, exist_ok=True)

    for item in final_results:
        source_component = item['sourceComponent']
        destination_component = item['destinationComponent']
        result = item['result']

        # YAML structure initialization
        yaml_content = {
            'apiVersion': 'networking.istio.io/v1alpha3',
            'kind': 'EnvoyFilter',
            'metadata': {
                'name': f'{source_component}-filter',
                'namespace': namespace
            },
            'spec': {
                'workloadSelector': {
                    'labels': {
                        'app': source_component
                    }
                },
                'configPatches': [
                    {
                        'applyTo': 'HTTP_FILTER',
                        'match': {
                            'context': 'SIDECAR_OUTBOUND',
                            'listener': {
                                'filterChain': {
                                    'filter': {
                                        'name': 'envoy.filters.network.http_connection_manager'
                                    }
                                }
                            }
                        },
                        'patch': {
                            'operation': 'INSERT_BEFORE',
                            'value': {
                                'name': 'envoy.filters.http.lua',
                                'typed_config': {
                                    '@type': 'type.googleapis.com/envoy.extensions.filters.http.lua.v3.Lua',
                                    'inline_code': ''  # Lua code will be inserted here
                                }
                            }
                        }
                    }
                ]
            }
        }

        # Generate Lua code
        lua_code = generate_lua_code(result, destination_component)

        # Wrap lua_code with LiteralUnicode to preserve formatting
        yaml_content['spec']['configPatches'][0]['patch']['value']['typed_config']['inline_code'] = LiteralUnicode(lua_code)

        # Save the YAML content to a file in the correct folder structure
        file_name = os.path.join(base_folder, f"{source_component}-filter.yaml")
        with open(file_name, 'w') as yaml_file:
            yaml.dump(yaml_content, yaml_file, sort_keys=False, default_flow_style=False)

def generate_lua_code(result, destination_component):
    lua_lines = [
        'local pod_ip = nil',
        'local counters = {}',  # Array to dynamically store counters
        'local status_value = 0',     # To check status >= 400
        f'local required_vars = {len(result)}',  # Number of unique IPs (replicas)         
        '',               
        'for i = 1, required_vars do',
        '  if not counters[i] then',
        '    counters[i] = 0',  # Initialize each counter to 0',
        '  end',
        'end',
        '',
        'function envoy_on_request(request_handle)',
        '  if not pod_ip then',
        '    local handle = io.popen("hostname -i")',
        '    pod_ip = handle:read("*a"):match("^%s*(.-)%s*$")',
        '    handle:close()',
        '  end',
        '',
        '  local destination = request_handle:headers():get(":authority")',
        '  local domain = destination:match("^([^:]+)")',
        '',
        f'  if domain == "{destination_component}" and status_value == 0 then',
        '    local new_destination = nil',
    ]

    # Start of the per-sourceReplica Lua code
    for index, (source_replica, data) in enumerate(result.items(), 1):
        source_ip = data['sourceReplicaIP']
        allocation_sequence = data['allocationSequence']

        lua_lines.append(f'    if pod_ip == "{source_ip}" then')
        lua_lines.append(f'      counters[{index}] = counters[{index}] + 1')
        
        # Convert allocation_sequence to a Lua table literal
        sequence_elements = ', '.join(f'"{dest}"' for dest in allocation_sequence)
        lua_lines.append(f'      local sequence = {{ {sequence_elements} }}')

        lua_lines.append(f'      if counters[{index}] > #sequence then')
        lua_lines.append(f'        counters[{index}] = 1')
        lua_lines.append('      end')
        lua_lines.append(f'      new_destination = sequence[counters[{index}]]')
        lua_lines.append('    end')

    lua_lines.append('    if new_destination then')
    lua_lines.append('      local new_destination_with_port = new_destination .. destination:match("(:.*)$")')
    lua_lines.append('      request_handle:headers():replace(":authority", new_destination_with_port)')
    lua_lines.append('      request_handle:headers():replace("Host", new_destination_with_port)')
    lua_lines.append('    end')
    lua_lines.append('  end')
    lua_lines.append('end')
    lua_lines.append('')
    lua_lines.append('function envoy_on_response(response_handle)')
    lua_lines.append('  local status_code = tonumber(response_handle:headers():get(":status"))')
    lua_lines.append('')
    lua_lines.append('  if status_code >= 400 then')
    lua_lines.append('    status_value = 1')
    lua_lines.append('  end')
    lua_lines.append('end')

    lua_code = '\n'.join(lua_lines)
    return lua_code

def main():
    # Replace with your actual URL
    url = "http://localhost:21002/traffic-sched?value=15&namespace=pair&algorithm=hk"

    # Extract namespace from URL query parameters
    parsed_url = urlparse.urlparse(url)
    query_params = parse_qs(parsed_url.query)
    namespace = query_params.get("namespace", [None])[0]

    # Fetch the JSON data
    json_data = fetch_json_data(url)

    # Extract the 'final_results' part
    final_results = json_data.get('final_results', [])
    
    # Generate YAML files based on 'final_results' and dynamic namespace
    generate_yaml_files(final_results, namespace)

if __name__ == "__main__":
    main()
