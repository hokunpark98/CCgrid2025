from collections import defaultdict, deque
import json

def allocate_slots_smooth_weighted(counts):
    """필요 시 계속 순환할 수 있는 슬롯 리스트 생성"""
    total_slots = sum(counts)
    num_types = len(counts)
    slots = []
    current_weights = [0] * num_types
    total_weight = sum(counts)
    
    while len(slots) < total_slots:
        # 각 타입의 현재 가중치를 업데이트
        for i in range(num_types):
            current_weights[i] += counts[i]
        
        # 가장 높은 현재 가중치를 가진 타입 선택
        selected = current_weights.index(max(current_weights))
        slots.append(selected)  # 타입 인덱스를 슬롯에 추가
        
        # 선택된 타입의 현재 가중치를 전체 가중치 합만큼 감소
        current_weights[selected] -= total_weight
    
    return slots

def generate_replica_sequence(traffic_results):
    sequence_mapping = {}

    # 각 상위 레플리카에 대해 순서를 생성
    for source_replica, targets in traffic_results.items():
        counts = [traffic for traffic in targets.values()]
        
        # 필요한 만큼 순환하도록 슬롯을 생성
        order_sequence = allocate_slots_smooth_weighted(counts)
        
        target_keys = list(targets.keys())
        
        # 하위 레플리카 매칭을 위해 sequence에 따라 할당
        mapped_sequence = deque(target_keys[slot] for slot in order_sequence)
        
        # JSON 형식으로 매핑 결과를 저장
        sequence_mapping[source_replica] = list(mapped_sequence)
    
    return sequence_mapping

def process_traffic_allocation(traffic_results_json):
    # traffic_results만 추출하여 작업 수행
    traffic_results = traffic_results_json.get("traffic_results", {})

    # 각 상위 레플리카에 대해 트래픽 비율 기반의 할당 순서 생성
    sequence_mapping = {}
    for component_pair, source_replicas in traffic_results.items():
        sequence_mapping[component_pair] = generate_replica_sequence(source_replicas)
    
    # 최종 JSON 형식으로 결과 반환
    result_json = {
        "traffic_allocation_sequences": sequence_mapping
    }
    return json.dumps(result_json, indent=4)

# 예시 JSON 데이터
traffic_results_json = {
    "deficient_edge_counts": {
        "frontend-pair->trigono": {
            "trigono-worker4-6857596fdb-6bngq": [
                [
                    "frontend-pair-worker2-8569d4669f-jcvfv",
                    "trigono-worker4-6857596fdb-6bngq",
                    13
                ],
                [
                    "frontend-pair-worker3-655c586b5d-qct2w",
                    "trigono-worker4-6857596fdb-6bngq",
                    2
                ]
            ],
            "trigono-worker5-56bd89775-cnj76": [
                [
                    "frontend-pair-worker1-9656c64c6-qw76l",
                    "trigono-worker5-56bd89775-cnj76",
                    13
                ],
                [
                    "frontend-pair-worker2-8569d4669f-jcvfv",
                    "trigono-worker5-56bd89775-cnj76",
                    5
                ]
            ]
        },
        "load-generator->frontend-pair": {}
    },
    "traffic_results": {
        "frontend-pair->trigono": {
            "frontend-pair-worker1-9656c64c6-qw76l": {
                "trigono-worker1-1-655b665c6c-jzbgg": 26,
                "trigono-worker5-56bd89775-cnj76": 13
            },
            "frontend-pair-worker2-8569d4669f-jcvfv": {
                "trigono-worker2-568dc6455b-8ctwj": 26,
                "trigono-worker4-6857596fdb-6bngq": 13,
                "trigono-worker5-56bd89775-cnj76": 5
            },
            "frontend-pair-worker3-655c586b5d-qct2w": {
                "trigono-worker3-599978479c-gzb6f": 15,
                "trigono-worker4-6857596fdb-6bngq": 2
            }
        },
        "load-generator->frontend-pair": {
            "load-generator-worker1-759888dfc9-c7jdp": {
                "frontend-pair-worker1-9656c64c6-qw76l": 39,
                "frontend-pair-worker2-8569d4669f-jcvfv": 39,
                "frontend-pair-worker3-655c586b5d-qct2w": 22
            }
        }
    }
}

# 결과 호출
print(process_traffic_allocation(traffic_results_json))
