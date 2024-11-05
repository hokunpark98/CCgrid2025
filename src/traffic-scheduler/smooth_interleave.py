from collections import defaultdict, deque
import json

def allocate_slots_smooth_weighted(counts):
    total_slots = sum(counts)
    num_types = len(counts)
    slots = []
    current_weights = [0] * num_types
    total_weight = sum(counts)
    
    while len(slots) < total_slots:
        for i in range(num_types):
            current_weights[i] += counts[i]
        
        selected = current_weights.index(max(current_weights))
        slots.append(selected)
        current_weights[selected] -= total_weight
    
    return slots

def extract_ips(metrics_data):
    """metrics_data에서 각 pod 이름에 대한 IP 주소를 추출"""
    replica_ips = {}
    for component in metrics_data["Components"]:
        for pod in component["Replicas"]:
            replica_ips[pod["Replica"]] = pod["IP"]
    return replica_ips

def generate_replica_sequence(traffic_results, replica_ips):
    result = {}

    # 각 상위 레플리카에 대해 순서를 생성
    for source_replica, targets in traffic_results.items():
        counts = [traffic for traffic in targets.values()]
        order_sequence = allocate_slots_smooth_weighted(counts)
        target_keys = list(targets.keys())
        
        # 하위 레플리카 매칭을 위해 sequence에 따라 할당
        mapped_sequence = [target_keys[slot] for slot in order_sequence]
        
        # source_replica에 대해 source_ip 포함한 매핑 생성
        source_ip = replica_ips.get(source_replica, "IP_NOT_FOUND")
        result[source_replica] = {
            "sourceReplicaIP": source_ip,
            "allocationSequence": mapped_sequence
        }
    
    return result

def process_traffic_allocation(traffic_results, metrics_data):
    replica_ips = extract_ips(metrics_data)
    final_results = []

    for component_pair, source_replicas in traffic_results.items():
        source, destination = component_pair.split("->")
        allocation_data = generate_replica_sequence(source_replicas, replica_ips)

        final_results.append({
            "sourceComponent": source,
            "destinationComponent": destination,
            "result": allocation_data
        })
    
    return final_results