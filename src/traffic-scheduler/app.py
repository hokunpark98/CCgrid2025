from flask import Flask, request, jsonify
from algorithm import *
from smooth_interleave import *

app = Flask(__name__)

@app.route('/traffic-sched', methods=['GET'])
def traffic_scheduling():
    # Retrieve query parameters
    value = request.args.get('value')
    namespace = request.args.get('namespace')
    algorithm = request.args.get('algorithm', 'default')  # Default algorithm can be updated later

    # Validate required parameters
    if not value or not namespace:
        return jsonify({'error': 'Missing required parameters'}), 400

    # Construct URLs for fetching DAG and metrics data
    dag_url = f"http://10.96.9.243:21001/dag?value={value}&namespace={namespace}"
    metrics_url = f"http://10.96.9.243:21001/metrics?value={value}&namespace={namespace}"

    try:
        # 데이터 가져오기
        dag_data, metrics_data = fetch_data(dag_url, metrics_url)

        # 그래프 생성
        G, component_graph, component_replicas = build_graphs(dag_data, metrics_data)

        # 루트 컴포넌트 찾기
        root_components = find_root_components(component_graph)

        # 트래픽 수신량 및 용량 초기화
        traffic_received, traffic_capacity = initialize_traffic(component_replicas, root_components)

        # 트래픽 할당 수행
        traffic_results, deficient_edge_counts = perform_traffic_allocation(
            G, component_graph, component_replicas, traffic_received, traffic_capacity
        )

        final_results = process_traffic_allocation(traffic_results, metrics_data)
        #print_results(traffic_results, deficient_edge_counts)

        # Return the results in JSON format
        return jsonify({
            'traffic_results': traffic_results,
            'deficient_edge_counts': deficient_edge_counts,
            'final_results': final_results,
            'metrics_data': metrics_data
        })

    except Exception as e:
        # Error handling
        return jsonify({'error': str(e)}), 500

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=21002)
