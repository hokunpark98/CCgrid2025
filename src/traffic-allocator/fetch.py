import requests

def fetch_json_data(url):
    response = requests.get(url)
    response.raise_for_status()
    return response.json()
