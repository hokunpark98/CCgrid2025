import requests
import networkx as nx
from collections import defaultdict
from yaml_manager import *
from concurrent.futures import ThreadPoolExecutor
import copy

def find_root_components(component_graph):
    return [node for node in component_graph.nodes() if component_graph.in_degree(node) == 0]

def fetch_data(dag_url, metrics_url):
    dag_data = requests.get(dag_url).json()
    metrics_data = requests.get(metrics_url).json()
    return dag_data, metrics_data

def build_graphs(dag_data, metrics_data):
    G = nx.DiGraph()
    component_replicas = {}

    for component in metrics_data["Components"]:
        comp_name = component["Component"]
        replicas = component["Replicas"]
        component_replicas[comp_name] = {}

        for replica in replicas:
            replica_name = replica["ReplicaVersion"]
            replica["Frequency"] = float(replica["Frequency"])
            replica["node"] = replica["Worker"]
            G.add_node(replica_name, **replica, component=comp_name)
            component_replicas[comp_name][replica_name] = replica

    component_graph = nx.DiGraph()
    for link in dag_data:
        src_comp = link["Source"]
        dst_comp = link["Destination"]
        component_graph.add_edge(src_comp, dst_comp)
        for src_replica in component_replicas.get(src_comp, {}):
            for dst_replica in component_replicas.get(dst_comp, {}):
                G.add_edge(src_replica, dst_replica)

    return G, component_graph, component_replicas

def update_istio_configs(traffic_results, component_replicas, namespace, vsdrpath, custom_objects_api):
    """트래픽 결과 기반으로 VirtualService와 DestinationRule 업데이트"""
    service_routes = []

    print('traffic_results' , type(traffic_results))

    for comp_pair, allocations in traffic_results.items():
        print('comp_pair', comp_pair)
        src_comp, dst_comp = comp_pair.split("->")
        for up_replica, down_allocations in allocations.items():
            total_out = sum(down_allocations.values())
            if total_out == 0:
                continue
            for down_replica, val in down_allocations.items():
                weight = int(round((val / total_out) * 100))
                if weight > 0:
                    src_version = component_replicas[src_comp][up_replica]["ReplicaVersion"]
                    dst_version = component_replicas[dst_comp][down_replica]["ReplicaVersion"]
                    service_routes.append((src_comp, src_version, dst_comp, dst_version, weight))

    print('service_routes', service_routes)
    apply_virtual_service(custom_objects_api, service_routes, namespace, vsdrpath)

    service_versions = {}
    for comp_name, replicas in component_replicas.items():
        versions = set(replica["ReplicaVersion"] for replica in replicas.values())
        service_versions[comp_name] = versions

    apply_destination_rules(namespace, custom_objects_api, service_versions, vsdrpath)


def allocate_by_ratio(replicas, total=100):
    """
    replicas: {replica_name: {"Frequency": f, ...}, ...}
    total: 정규화 기준 (기본 100)
    
    초기 Frequency 합이 어떠한 값이든 그 합에 비례하여 total을 기준으로 비율 재계산.
    예: Frequency가 {b1:2, b2:1}이면 합3. 
    이때 total=100이라면 b1= (2/3)*100=66.7≈67, b2=33.3≈33
    """
    if not replicas:
        return {}
    sum_val = sum(r["Frequency"] for r in replicas.values())
    if sum_val == 0:
        return {r: 0 for r in replicas}

    raw_allocations = {r: (replicas[r]["Frequency"] / sum_val) * total for r in replicas}
    int_allocations = {r: int(round(val)) for r, val in raw_allocations.items()}
    allocated_sum = sum(int_allocations.values())
    diff = total - allocated_sum
    if diff != 0 and int_allocations:
        min_rep = min(int_allocations, key=int_allocations.get)
        int_allocations[min_rep] += diff
    return int_allocations

def initialize_traffic(component_replicas, root_components, initial_traffic=100):
    """
    root component(servicea)는 항상 total 100 할당.
    나머지 컴포넌트는 frequency 비율에 따라 100 할당.
    """
    traffic_capacity = {}

    for component, replicas in component_replicas.items():
        if component in root_components:
            equal_share = int(initial_traffic / len(replicas))
            allocated = {r: equal_share for r in replicas.keys()}
            diff = initial_traffic - sum(allocated.values())
            if diff != 0:
                first_rep = next(iter(allocated))
                allocated[first_rep] += diff
            traffic_capacity.update(allocated)
        else:
            traffic_capacity.update(allocate_by_ratio(replicas, initial_traffic))

    return traffic_capacity

def greedy_pair_allocation(upstream_replicas, downstream_replicas, traffic_capacity):
    traffic_matrix = defaultdict(dict)
    temp_traffic_capacity = copy.deepcopy(traffic_capacity)

    # Step 1: 같은 노드 할당
    for down in downstream_replicas:
        needed = temp_traffic_capacity.get(down, 0)
        if needed <= 0:
            continue
        same_node_ups = [up for up in upstream_replicas if upstream_replicas[up]["node"] == downstream_replicas[down]["node"]]
        same_node_ups.sort(key=lambda x: temp_traffic_capacity.get(x, 0), reverse=True)

        for up in same_node_ups:
            if needed <= 0:
                break
            assignable = min(needed, temp_traffic_capacity.get(up, 0))
            if assignable > 0:
                traffic_matrix[up][down] = traffic_matrix[up].get(down, 0) + assignable
                temp_traffic_capacity[up] -= assignable
                needed -= assignable
                temp_traffic_capacity[down] -= assignable

    print('traffic_capacity', temp_traffic_capacity)
    
    # Step 2: 다른 노드 할당
    still_needed = [(d, temp_traffic_capacity[d]) for d in downstream_replicas if temp_traffic_capacity.get(d, 0) > 0]
    still_needed.sort(key=lambda x: x[1], reverse=True)
    still_supply = [(u, temp_traffic_capacity[u]) for u in upstream_replicas if temp_traffic_capacity[u] > 0]
    still_supply.sort(key=lambda x: x[1], reverse=True)

    print('still_needed', still_needed)
    print('still_supply', still_supply)
    for d, need_amt in still_needed:
        amt_needed = need_amt
        for i, (u, sup_amt) in enumerate(still_supply):
            if amt_needed <= 0:
                break
            if sup_amt <= 0:
                continue
            assignable = min(amt_needed, sup_amt)
            traffic_matrix[u][d] = traffic_matrix[u].get(d, 0) + assignable
            amt_needed -= assignable
            sup_amt -= assignable
            temp_traffic_capacity[u] -= assignable
            temp_traffic_capacity[d] -= assignable
            still_supply[i] = (u, sup_amt)

    return traffic_matrix

def perform_traffic_allocation(G, component_graph, component_replicas, traffic_capacity, namespace):
    traffic_results = {}

    components_in_order = list(nx.topological_sort(component_graph))

    def process_pair(component, dst_comp):        
        upstream_reps = component_replicas[component]
        downstream_reps = component_replicas[dst_comp]
        pair_alloc = greedy_pair_allocation(upstream_reps, downstream_reps, traffic_capacity)
        return (f"{component}->{dst_comp}", pair_alloc)

    futures = []
    with ThreadPoolExecutor() as executor:
        for component in components_in_order:
            downstream_components = list(component_graph.successors(component))
            for dst_comp in downstream_components:
                futures.append(executor.submit(process_pair, component, dst_comp))

        for future in futures:
            comp_pair, allocation = future.result()
            traffic_results[comp_pair] = allocation

   
    return traffic_results, component_replicas, namespace
