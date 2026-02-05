import requests
import json
import time

BASE_URL = "http://localhost:8888"

def test_login():
    print("Testing Login...")
    resp = requests.post(f"{BASE_URL}/ipa/apis/json/general/login", json={"username": "test", "password": "pwd"})
    data = resp.json()
    assert data["result"] == "SUCCESS"
    assert "sessionId" in data
    assert "jwtToken" in data
    print("Login Passed")

def test_get_provisioning_data():
    print("Testing GetProvisioningData...")
    resp = requests.post(f"{BASE_URL}/ipa/apis/json/provisioning/getProvisioningData", json={
        "username": "test", "password": "pwd", "start": 0, "limit": 10
    })
    data = resp.json()
    assert data["result"] == "SUCCESS"
    assert "fieldNames" in data
    assert "PREPAID_DATA_BALANCE" in data["fieldNames"]
    assert len(data["data"]) > 0
    sim = data["data"][0]
    assert "PREPAID_DATA_BALANCE" in sim
    assert sim["SIM_STATUS_CHANGE"] in ["Activated", "Suspended", "Terminated", "Pre-Activated"]
    print("GetProvisioningData Passed")

def test_update_status():
    print("Testing UpdateProvisioningData (Status)...")
    # First get a SIM
    resp = requests.post(f"{BASE_URL}/ipa/apis/json/provisioning/getProvisioningData", json={"username": "test", "password": "pwd", "limit": 1})
    cli = resp.json()["data"][0]["CLI"]
    
    # Update to Suspended
    payload = {
        "username": "test", "password": "pwd",
        "actions": [{
            "actionType": "SIM_STATE_CHANGE",
            "targetValue": "Suspended",
            "targetId": "",
            "subscribers": [{"neId": cli}]
        }]
    }
    resp = requests.post(f"{BASE_URL}/ipa/apis/json/provisioning/updateProvisioningData", json=payload)
    assert resp.json()["result"] == "SUCCESS"
    
    # Verify
    resp = requests.post(f"{BASE_URL}/ipa/apis/json/provisioning/getProvisioningData", json={
        "username": "test", "password": "pwd",
        "search": [{"fieldName": "CLI", "fieldValue": cli}]
    })
    sim = resp.json()["data"][0]
    assert sim["SIM_STATUS_CHANGE"] == "Suspended"
    print("UpdateStatus Passed")

if __name__ == "__main__":
    try:
        test_login()
        test_get_provisioning_data()
        test_update_status()
        print("\nAll Simulator Tests Passed!")
    except Exception as e:
        print(f"\nTest Failed: {e}")
        exit(1)
