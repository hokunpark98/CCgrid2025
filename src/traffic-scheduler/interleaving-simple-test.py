def allocate_slots(counts):
    total_slots = sum(counts)
    # 초기 슬롯을 빈 문자열로 설정
    slots = [''] * total_slots
    total_count = sum(counts)  # 전체 개수
    num_types = len(counts)  # 숫자의 종류 수

    # 각 숫자에 대한 간격 계산
    intervals = [total_slots // (count + 1) if count > 0 else 0 for count in counts]

    # 각 숫자를 비율에 맞게 배치
    indices = [0] * num_types  # 각 숫자의 현재 인덱스
    for i in range(total_count):
        selected = -1
        for j in range(num_types):
            if counts[j] > 0:
                if selected == -1 or indices[j] < indices[selected] + intervals[selected]:
                    selected = j

        if selected != -1:
            # 숫자를 배치
            while indices[selected] < total_slots and slots[indices[selected]] != '':
                indices[selected] += 1  # 비어 있는 슬롯 찾기
            
            if indices[selected] < total_slots:  # 빈 슬롯이 남아 있다면
                slots[indices[selected]] = str(selected + 1)  # 숫자 배치
                counts[selected] -= 1  # 남은 개수 줄이기

            # 다음 슬롯으로 이동
            indices[selected] += intervals[selected] + 1  # 간격을 넓히기 위해 +1 추가

    # 최종적으로 남은 슬롯을 채우기
    for i in range(total_slots):
        if slots[i] == '':
            for j in range(num_types):
                if counts[j] > 0:
                    slots[i] = str(j + 1)  # 숫자 배치 (1부터 시작)
                    counts[j] -= 1
                    break

    return ''.join(slots)

# 예시: 1의 개수 29, 2의 개수 10, 3의 개수 5
result = allocate_slots([29, 10])
print(result)
