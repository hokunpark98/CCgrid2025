def allocate_slots_smooth_weighted(counts):
    total_slots = sum(counts)
    num_types = len(counts)
    slots = []
    current_weights = [0] * num_types
    total_weight = sum(counts)
    
    for _ in range(total_slots):
        # 각 타입의 현재 가중치를 업데이트
        for i in range(num_types):
            current_weights[i] += counts[i]
        
        # 가장 높은 현재 가중치를 가진 타입 선택
        selected = current_weights.index(max(current_weights))
        slots.append(str(selected + 1))  # 타입을 문자열로 변환하여 슬롯에 추가
        
        # 선택된 타입의 현재 가중치를 전체 가중치 합만큼 감소
        current_weights[selected] -= total_weight
    
    return ''.join(slots)

# 예시: 1의 개수 26, 2의 개수 13, 3의 개수 5
result = allocate_slots_smooth_weighted([26, 13, 5])
print(result)
