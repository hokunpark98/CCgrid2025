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

# 루트 노드에 초기 트래픽 할당 (루트 노드를 loadbalancer로 둬야할 듯)
for component in root_components:
    for replica_name in component_replicas[component]:
        traffic_received[replica_name] = 100  # 필요한 경우 실제 값으로 수정

# 트래픽 할당 함수 정의 (위에서 수정한 함수 사용)
# 자기들 내부에서 값이니까 예측된 값 총합에다가 자신의 예측된 값으로 해서 capa구하면 문제는 없을듯
def allocate_traffic(upstream_replicas, downstream_replicas, traffic_received):
    # 다운스트림 포드의 용량 계산 (정수 연산)
    total_freq = sum(replica_attrs["Frequency"] for replica_attrs in downstream_replicas.values())
    downstream_capacity = {}
    for replica_name, attrs in downstream_replicas.items():
        capacity = int(round((attrs["Frequency"] / total_freq) * 100))
        downstream_capacity[replica_name] = capacity

    # 업스트림 포드의 남은 용량 (정수 연산)
    upstream_capacity = {up_replica: int(traffic_received.get(up_replica, 0)) for up_replica in upstream_replicas}

    # 최대 매칭을 찾기 위해 이분 그래프 생성
    bipartite_graph = nx.Graph()
    bipartite_graph.add_nodes_from(upstream_replicas, bipartite=0)
    bipartite_graph.add_nodes_from(downstream_replicas.keys(), bipartite=1)

    # 업스트림과 다운스트림 포드 간의 간선 추가 (용량이 있는 경우에만)
    for up_replica in upstream_replicas:
        for down_replica in downstream_replicas:
            # 업스트림 포드와 다운스트림 포드 간의 가능한 최대 트래픽 (정수 연산)
            max_possible = min(upstream_capacity[up_replica], downstream_capacity[down_replica])
            if max_possible > 0:
                bipartite_graph.add_edge(up_replica, down_replica)

    # 최대 매칭 찾기
    matching = nx.algorithms.bipartite.maximum_matching(bipartite_graph, top_nodes=upstream_replicas)

    # 매칭 결과에 따라 트래픽 할당
    traffic_matrix = {up_replica: {} for up_replica in upstream_replicas}
    assigned_downstreams = set()
    for up_replica in upstream_replicas:
        if up_replica in matching:
            down_replica = matching[up_replica]
            # 트래픽 할당 (정수 연산)
            assignable = min(upstream_capacity[up_replica], downstream_capacity[down_replica])
            traffic_matrix[up_replica][down_replica] = assignable
            upstream_capacity[up_replica] -= assignable
            downstream_capacity[down_replica] -= assignable
            assigned_downstreams.add(down_replica)

    # 부족 간선 목록 초기화
    deficient_edges = {replica_name: [] for replica_name in downstream_replicas}

    # 남은 업스트림 및 다운스트림 포드
    remaining_upstream = [up for up in upstream_replicas if upstream_capacity[up] > 0]
    remaining_downstream = [down for down in downstream_replicas if downstream_capacity[down] > 0 and down not in assigned_downstreams]

    # 부족 간선 총합 최소화를 위한 추가 할당 (정수 연산)
    while remaining_upstream and remaining_downstream:
        # 업스트림 포드와 다운스트림 포드 정렬
        remaining_upstream.sort(key=lambda x: -upstream_capacity[x])  # 남은 트래픽이 많은 순
        remaining_downstream.sort(key=lambda x: (len(deficient_edges[x]), -downstream_capacity[x]))  # 부족 간선 수 적은 순, 남은 용량 많은 순

        for up_replica in remaining_upstream:
            up_remain = upstream_capacity[up_replica]
            if up_remain <= 0:
                continue
            for down_replica in remaining_downstream:
                down_remain = downstream_capacity[down_replica]
                if down_remain <= 0:
                    continue
                assignable = min(up_remain, down_remain)
                if assignable > 0:
                    if down_replica not in traffic_matrix[up_replica]:
                        traffic_matrix[up_replica][down_replica] = 0
                    traffic_matrix[up_replica][down_replica] += assignable
                    upstream_capacity[up_replica] -= assignable
                    downstream_capacity[down_replica] -= assignable
                    # 부족 간선으로 기록
                    deficient_edges[down_replica].append((up_replica, down_replica, assignable))
                    up_remain -= assignable
                if up_remain <= 0:
                    break

        # 남은 업스트림 및 다운스트림 포드 업데이트
        remaining_upstream = [up for up in remaining_upstream if upstream_capacity[up] > 0]
        remaining_downstream = [down for down in remaining_downstream if downstream_capacity[down] > 0]

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
        if edges:
            print(f"  {down_replica}의 부족 간선 수: {len(edges)}")
            for edge_info in edges:
                up_replica, down_replica, traffic = edge_info
                print(f"    부족 간선 - {up_replica} -> {down_replica}: 할당된 트래픽 {traffic}")
        else:
            print(f"  {down_replica}의 부족 간선 수: 0")
