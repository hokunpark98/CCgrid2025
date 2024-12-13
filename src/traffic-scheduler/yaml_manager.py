import yaml
import os
from kubernetes.client.exceptions import ApiException

def apply_yaml(custom_objects_api, yaml_content):
    yaml_string = yaml.dump(yaml_content, default_flow_style=False)
    yaml_object = yaml.safe_load(yaml_string)
    group, version = yaml_object["apiVersion"].split('/')
    plural = yaml_object["kind"].lower() + "s"
    name = yaml_object["metadata"]["name"]
    namespace = yaml_object.get("metadata", {}).get("namespace", "default")

    try:
        custom_objects_api.create_namespaced_custom_object(
            group=group,
            version=version,
            namespace=namespace,
            plural=plural,
            body=yaml_object
        )
    except ApiException as e:
        if e.status == 409:
            existing_object = custom_objects_api.get_namespaced_custom_object(
                group=group,
                version=version,
                namespace=namespace,
                plural=plural,
                name=name
            )
            yaml_object["metadata"]["resourceVersion"] = existing_object["metadata"]["resourceVersion"]
            custom_objects_api.replace_namespaced_custom_object(
                group=group,
                version=version,
                namespace=namespace,
                plural=plural,
                name=name,
                body=yaml_object
            )
        else:
            raise

def create_virtual_service_yaml(service_routes, virtualservice_dir, namespace):
    grouped_routes = {}
    for route in service_routes:
        source, source_version, destination, destination_version, weight = route
        key = (source, source_version)
        if key not in grouped_routes:
            grouped_routes[key] = {}
        if destination not in grouped_routes[key]:
            grouped_routes[key][destination] = {}
        grouped_routes[key][destination][destination_version] = weight

    yamls = {}
    for (source, source_version), destinations in grouped_routes.items():
        for destination, version_weights in destinations.items():
            file_path = os.path.join(virtualservice_dir, f"{source}-to-{destination}-vs.yaml")

            routes_yaml = []
            for version, weight in version_weights.items():
                routes_yaml.append({
                    'destination': {
                        'host': destination,
                        'subset': version
                    },
                    'weight': weight
                })

            match_entry = {
                'match': [
                    {
                        'sourceLabels': {
                            'app': source,
                            'version': source_version
                        }
                    }
                ],
                'route': routes_yaml
            }

            if os.path.exists(file_path):
                with open(file_path, 'r') as file:
                    existing_content = yaml.safe_load(file)

                # source_version 기반으로 route 업데이트/추가
                updated = False
                for http_entry in existing_content['spec']['http']:
                    match = http_entry['match'][0]
                    if match['sourceLabels']['version'] == source_version:
                        http_entry['route'] = routes_yaml
                        updated = True
                        break

                if not updated:
                    existing_content['spec']['http'].append(match_entry)

                yaml_content = existing_content
            else:
                yaml_content = {
                    'apiVersion': 'networking.istio.io/v1alpha3',
                    'kind': 'VirtualService',
                    'metadata': {
                        'name': f'{source}-to-{destination}-vs',
                        'namespace': namespace
                    },
                    'spec': {
                        'hosts': [destination],
                        'http': [match_entry]
                    }
                }

            yamls[(source, source_version)] = yaml_content

            with open(file_path, 'w') as file:
                yaml.dump(yaml_content, file, default_flow_style=False)

    return yamls

def create_destination_rule_yaml(service, versions, namespace):
    subsets = ""
    for version in versions:
        subsets += f"""
  - name: {version}
    labels:
      version: {version}"""
    
    yaml_content = f"""
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: {service}-dr
  namespace: {namespace}
spec:
  host: {service}
  subsets:{subsets}
  trafficPolicy:
    loadBalancer:
      simple: LEAST_REQUEST
"""

    return yaml.safe_load(yaml_content)


def apply_virtual_service(custom_objects_api, service_routes, namespace, vsdrpath):
    # 기존 VS 파일 삭제
    for filename in os.listdir(vsdrpath):
        file_path = os.path.join(vsdrpath, filename)
        if os.path.isfile(file_path):
            try:
                os.remove(file_path)
            except FileNotFoundError:
                pass

    virtual_service_yamls = create_virtual_service_yaml(service_routes, vsdrpath, namespace)
    for vs_yaml in virtual_service_yamls.values():
        #apply_yaml(custom_objects_api, vs_yaml)
        file_path = os.path.join(vsdrpath, f"{vs_yaml['metadata']['name']}.yaml")
        with open(file_path, 'w') as file:
            yaml.dump(vs_yaml, file, default_flow_style=False)

def apply_destination_rules(namespace, custom_objects_api, service_versions, vsdrpath):
    for service, versions in service_versions.items():
        destination_rule_yaml = create_destination_rule_yaml(service, versions, namespace)
        #apply_yaml(custom_objects_api, destination_rule_yaml)
        file_path = os.path.join(vsdrpath, f"{service}-dr.yaml")
        with open(file_path, 'w') as file:
            yaml.dump(destination_rule_yaml, file, default_flow_style=False)
