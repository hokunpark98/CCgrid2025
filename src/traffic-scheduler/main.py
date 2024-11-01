import requests
import networkx as nx

# URL 설정
dagURL = "http://10.96.9.243:21001/dag?value=15&namespace=pair"
metricsURL = "http://10.96.9.243:21001/metrics?value=15&namespace=pair"

# 데이터 가져오기
dag_data = requests.get(dagURL).json()
metrics_data = requests.get(metricsURL).json()

# 그래프 생성
G = nx.DiGraph()
component_replicas = {}

for component in metrics_data["Components"]:
    comp_name = component["Component"]
    replicas = component["Replicas"]
    component_replicas[comp_name] = []
    for replica in replicas:
        replica_name = replica["Replica"]
        replica["Frequency"] = float(replica["Frequency"])  # 주파수를 float으로 변환
        G.add_node(replica_name, **replica, component=comp_name)
        component_replicas[comp_name].append(replica_name)

# 컴포넌트 그래프 생성
component_graph = nx.DiGraph()
for link in dag_data:
    src_comp = link["Source"]
    dst_comp = link["Destination"]
    component_graph.add_edge(src_comp, dst_comp)
    for src_replica in component_replicas.get(src_comp, []):
        for dst_replica in component_replicas.get(dst_comp, []):
            G.add_edge(src_replica, dst_replica)

# 루트 컴포넌트 찾기 (입력 간선이 없는 컴포넌트)
root_components = [node for node in component_graph.nodes() if component_graph.in_degree(node) == 0]

# 트래픽 수신량 초기화
traffic_received = {}
for component in component_replicas.keys():
    for replica_name in component_replicas[component]:
        traffic_received[replica_name] = 0

# 루트 노드에 초기 트래픽 할당 (예: 100)
for component in root_components:
    for replica_name in component_replicas[component]:
        traffic_received[replica_name] = 100  # 필요한 경우 실제 값으로 수정

# 트래픽 할당 함수 정의
def allocate_traffic(upstream_replicas, downstream_replicas, traffic_received):
    # 수정된 allocate_traffic 함수 (위에서 제공한 코드)

    # 다운스트림 포드의 실제 용량 계산 (주파수 비율 기반)
    total_freq = sum(replica_attrs["Frequency"] for replica_attrs in downstream_replicas.values())
    downstream_capacity = {}
    for replica_name, attrs in downstream_replicas.items():
        capacity = int(round((attrs["Frequency"] / total_freq) * 100))  # 비율을 정수로 변환
        downstream_capacity[replica_name] = capacity

    # 다운스트림 용량의 총합이 정확히 100이 되도록 조정
    total_ratio = sum(downstream_capacity.values())
    difference = 100 - total_ratio
    if difference != 0:
        # 가장 높은 비율을 가진 포드에 조정값 추가
        max_replica = max(downstream_capacity, key=downstream_capacity.get)
        downstream_capacity[max_replica] += difference

    # 업스트림 포드의 남은 용량
    upstream_capacity = {up_replica: traffic_received.get(up_replica, 0) for up_replica in upstream_replicas}

    # 부족 간선 수 초기화
    deficient_edges = {replica_name: [] for replica_name in downstream_replicas}

    # 트래픽 매트릭스 초기화
    traffic_matrix = {up_replica: {} for up_replica in upstream_replicas}

    # 전체 트래픽 총량 확인
    total_upstream_capacity = sum(upstream_capacity.values())
    total_downstream_capacity = sum(downstream_capacity.values())

    # 업스트림 트래픽과 다운스트림 용량이 동일한지 확인
    if total_upstream_capacity != total_downstream_capacity:
        # 총량을 맞추기 위해 비율을 조정
        scale_factor = total_downstream_capacity / total_upstream_capacity
        for up_replica in upstream_capacity:
            upstream_capacity[up_replica] = int(upstream_capacity[up_replica] * scale_factor)

    # 업스트림 포드와 다운스트림 포드 간의 매트릭스 생성
    # 각 업스트림 포드에서 다운스트림 포드로 보낼 수 있는 최대 트래픽 계산
    possible_edges = []
    for up_replica in upstream_replicas:
        for down_replica in downstream_replicas:
            max_possible = min(traffic_received[up_replica], downstream_capacity[down_replica])
            possible_edges.append((up_replica, down_replica, max_possible))

    # 가능한 간선을 트래픽 양의 내림차순으로 정렬
    possible_edges.sort(key=lambda x: -x[2])

    # 간선 할당
    for up_replica, down_replica, max_possible in possible_edges:
        up_remain = upstream_capacity[up_replica]
        down_remain = downstream_capacity[down_replica]
        assignable = min(up_remain, down_remain)
        if assignable <= 0:
            continue
        # 할당
        if down_replica not in traffic_matrix[up_replica]:
            traffic_matrix[up_replica][down_replica] = 0
        traffic_matrix[up_replica][down_replica] += assignable
        # 용량 업데이트
        upstream_capacity[up_replica] -= assignable
        downstream_capacity[down_replica] -= assignable
        # 부족 간선 여부 확인
        if assignable < max_possible:
            deficient_edges[down_replica].append((up_replica, down_replica, assignable))

    # 부족 간선 정보와 트래픽 매트릭스 반환
    return traffic_matrix, deficient_edges

# 트래픽 할당 결과 저장
traffic_results = {}
deficient_edge_counts = {}

# 컴포넌트 그래프의 위상 정렬
components_in_order = list(nx.topological_sort(component_graph))

# 위상 정렬된 순서로 트래픽 할당
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
            traffic_received
        )
        # 트래픽 수신량 업데이트
        for up_replica, allocations in traffic_alloc.items():
            for down_replica, traffic in allocations.items():
                # 상위 포드의 잔여 트래픽 감소
                traffic_received[up_replica] -= traffic
                if traffic_received[up_replica] < 0:
                    traffic_received[up_replica] = 0  # 음수 방지
                # 하위 포드의 수신 트래픽 증가
                traffic_received[down_replica] += traffic
        # 결과 저장
        traffic_results[f"{component}->{dst_comp}"] = traffic_alloc
        deficient_edge_counts[f"{component}->{dst_comp}"] = deficient_edges

# 결과 출력
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
        print(f"  {down_replica}의 부족 간선 수: {len(edges)}")
        for edge_info in edges:
            up_replica, down_replica, traffic = edge_info
            print(f"    부족 간선 - {up_replica} -> {down_replica}: 할당된 트래픽 {traffic}")
