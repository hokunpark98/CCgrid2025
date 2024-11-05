import requests
import networkx as nx
from collections import defaultdict
import copy


def find_root_components(component_graph):
    """루트 컴포넌트를 찾는 함수 (입력 간선이 없는 컴포넌트)"""
    return [node for node in component_graph.nodes() if component_graph.in_degree(node) == 0]

def fetch_data(dag_url, metrics_url):
    """데이터를 가져오는 함수"""
    dag_data = requests.get(dag_url).json()
    metrics_data = requests.get(metrics_url).json()
    return dag_data, metrics_data

def build_graphs(dag_data, metrics_data):
    """그래프를 생성하고 컴포넌트 및 레플리카 정보를 구축하는 함수"""
    G = nx.DiGraph()
    component_replicas = {}

    for component in metrics_data["Components"]:
        comp_name = component["Component"]
        replicas = component["Replicas"]
        component_replicas[comp_name] = {}

        for replica in replicas:
            replica_name = replica["Replica"]
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


def initialize_traffic(component_replicas, root_components, initial_traffic=100):
    """트래픽 수신량을 초기화하고 각 레플리카의 용량을 설정하는 함수 - 각 레플리카가 얼만큼의 트래픽을 송수신<즉, 송신 수신 양은 동일함. 수신한 만큼 송신함>할 수 있는지
       이 부분을 동균이형이 만들 CPU frequency와 CPU utilization에 따른 비율 그걸로 바꾸면 됨.
    """
    traffic_received = {replica_name: 0 for replicas in component_replicas.values() for replica_name in replicas}
    traffic_capacity = {}

    # 각 레플리카의 수신 용량을 주파수 비율로 설정
    for component, replicas in component_replicas.items():
        total_frequency = sum(replica["Frequency"] for replica in replicas.values())
        
        # 초기 할당 계산 (소수점 반올림)
        raw_allocations = {
            replica_name: (replica["Frequency"] / total_frequency) * initial_traffic
            for replica_name, replica in replicas.items()
        }
        
        # 정수 할당으로 변환
        int_allocations = {replica_name: int(round(value)) for replica_name, value in raw_allocations.items()}
        allocated_sum = sum(int_allocations.values())
        
        # 보정: 할당 총합이 정확히 initial_traffic이 되도록 조정
        difference = initial_traffic - allocated_sum
        if difference != 0:
            # 가장 작은 할당량을 가진 레플리카에 보정값 추가
            min_replica = min(int_allocations, key=int_allocations.get)
            int_allocations[min_replica] += difference

        # 결과 반영
        traffic_capacity.update(int_allocations)

    # 루트 컴포넌트에 초기 트래픽 설정
    for component in root_components:
        for replica_name in component_replicas[component]:
            traffic_received[replica_name] = initial_traffic

    return traffic_received, traffic_capacity


def same_node_allocation(upstream_replicas, downstream_replicas, traffic_received, traffic_capacity, fixed_capacity):
    """대역폭 사용 최소화를 위한 같은 노드에 우선적으로 트래픽 할당하는 부분"""
    traffic_matrix = defaultdict(dict)
    
    for up_replica in upstream_replicas:
        up_remain = traffic_received[up_replica]

        for down_replica in downstream_replicas:
            if upstream_replicas[up_replica]["node"] == downstream_replicas[down_replica]["node"]:
                assignable = min(up_remain, fixed_capacity[down_replica])
                if assignable > 0:
                    traffic_matrix[up_replica][down_replica] = assignable
                    traffic_received[up_replica] -= assignable
                    traffic_capacity[down_replica] -= assignable
                    traffic_received[down_replica] += assignable

    return traffic_matrix



def minimize_deficient_edges(upstream_replicas, downstream_replicas, traffic_received, traffic_capacity, fixed_capacity):
    """부족 간선 최소화를 위한 트래픽 할당"""
    traffic_matrix = defaultdict(dict)
    deficient_edges = defaultdict(list)
    
    remaining_up_replicas = [up for up in upstream_replicas if traffic_received[up] > 0]
    remaining_down_replicas = [down for down in downstream_replicas if traffic_capacity[down] > 0]

    while remaining_up_replicas and remaining_down_replicas:
        remaining_down_replicas.sort(
            key=lambda down: (len(deficient_edges[down]), -traffic_capacity[down])
        )
        remaining_up_replicas.sort(key=lambda up: -traffic_received[up])

        for up_replica in remaining_up_replicas:
            up_remain = traffic_received[up_replica]

            for down_replica in remaining_down_replicas:
                down_remain = traffic_capacity[down_replica]
                max_possible = fixed_capacity[down_replica]  # 고정값

                assignable = min(up_remain, down_remain)  # 전송할 수 있는 값

                if assignable > 0:
                    if down_replica not in traffic_matrix[up_replica]:
                        traffic_matrix[up_replica][down_replica] = 0
                    traffic_matrix[up_replica][down_replica] += assignable
                    traffic_received[up_replica] -= assignable
                    traffic_capacity[down_replica] -= assignable
                    traffic_received[down_replica] += assignable

                    if assignable < max_possible:  # 고정 값보다 작은 값을 전송하는 간선이 부족 간선
                        deficient_edges[down_replica].append((up_replica, down_replica, assignable))
                    
                    elif assignable == max_possible: # max_possible에 도달하면 풀 간선이기 때문에 부족간선 목록에서 제거함                     
                        if down_replica in deficient_edges:
                            deficient_edges.pop(down_replica, None)

                if traffic_received[up_replica] <= 0:
                    break

        remaining_up_replicas = [up for up in remaining_up_replicas if traffic_received[up] > 0]
        remaining_down_replicas = [down for down in remaining_down_replicas if traffic_capacity[down] > 0]

    return traffic_matrix, deficient_edges


def allocate_traffic(upstream_replicas, downstream_replicas, traffic_received, traffic_capacity):
    """트래픽을 할당하는 함수, 동일 노드 우선 할당 및 부족 간선 최소화 처리"""
    traffic_matrix = defaultdict(dict)
    deficient_edges = defaultdict(list)
    
    # 1. 각 업스트림 레플리카가 각 다운스트림 레플리카에 보낼 수 있는 고정 최대값 설정
    fixed_capacity = copy.deepcopy(traffic_capacity)

    # 2. 대역폭 사요을 최소화를 위한 동일 노드 간 우선 트래픽 할당
    same_node_traffic = same_node_allocation(upstream_replicas, downstream_replicas, traffic_received, traffic_capacity, fixed_capacity)
    for up_replica, down_replicas in same_node_traffic.items():
        for down_replica, amount in down_replicas.items():
            traffic_matrix[up_replica][down_replica] = amount

    # 3. 순간 트래픽 몰리는 것을 최소화 하기 위한 부족 간선 최소화 기반의 잔여 트래픽 할당
    remaining_traffic, remaining_deficient_edges = minimize_deficient_edges(
        upstream_replicas, downstream_replicas, traffic_received, traffic_capacity, fixed_capacity
    )
    for up_replica, down_replicas in remaining_traffic.items():
        for down_replica, amount in down_replicas.items():
            if down_replica not in traffic_matrix[up_replica]:
                traffic_matrix[up_replica][down_replica] = 0
            traffic_matrix[up_replica][down_replica] += amount
    
    # 남은 부족 간선 정보 업데이트
    deficient_edges.update(remaining_deficient_edges)

    return traffic_matrix, deficient_edges



def perform_traffic_allocation(G, component_graph, component_replicas, traffic_received, traffic_capacity):
    """트래픽 할당을 수행하는 함수"""
    traffic_results = {}
    deficient_edge_counts = {}

    components_in_order = list(nx.topological_sort(component_graph))

    for component in components_in_order:
        upstream_replicas = component_replicas[component]
        downstream_components = list(component_graph.successors(component))

        for dst_comp in downstream_components:
            downstream_replicas = component_replicas[dst_comp]
            upstream_replica_dict = {replica_name: G.nodes[replica_name] for replica_name in upstream_replicas}
            downstream_replica_dict = {replica_name: G.nodes[replica_name] for replica_name in downstream_replicas}

            # 트래픽 할당
            traffic_alloc, deficient_edges = allocate_traffic(
                upstream_replicas,
                downstream_replica_dict,
                traffic_received,
                traffic_capacity
            )

            # 결과 저장
            comp_pair = f"{component}->{dst_comp}"
            traffic_results[comp_pair] = traffic_alloc
            deficient_edge_counts[comp_pair] = deficient_edges

    return traffic_results, deficient_edge_counts


def print_results(traffic_results, deficient_edge_counts):
    """결과를 출력하는 함수"""
    print("전체 트래픽 할당 결과:")
    for comp_pair, allocation in traffic_results.items():
        print(f"\n{comp_pair}:")
        for up_replica, allocations in allocation.items():
            for down_replica, traffic in allocations.items():
                print(f"  {up_replica} -> {down_replica}: {traffic}")

    print("\n부족 간선 수 결과:")
    for comp_pair, deficiencies in deficient_edge_counts.items():
        print(f"\n{comp_pair}의 부족 간선 목록:")
        for down_replica, edges in deficiencies.items():
            if edges:
                print(f"  {down_replica}의 부족 간선 수: {len(edges)}")
                for edge_info in edges:
                    up_replica, down_replica, traffic = edge_info
                    print(f"    부족 간선 - {up_replica} -> {down_replica}: 할당된 트래픽 {traffic}")
            else:
                print(f"  {down_replica}의 부족 간선 수: 0")

