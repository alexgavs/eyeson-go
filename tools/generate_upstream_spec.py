import os
import json
import requests
import time
from datetime import datetime
import argparse

# Configuration
BASE_URL = os.getenv("EYESON_API_BASE_URL", "https://eot-portal.pelephone.co.il:8888")
USERNAME = os.getenv("EYESON_API_USERNAME", "")
PASSWORD = os.getenv("EYESON_API_PASSWORD", "")

if not USERNAME or not PASSWORD:
    print("Error: EYESON_API_USERNAME and EYESON_API_PASSWORD environment variables must be set.")
    # For now, allow running to generate structure even if auth fails, but warn.
    # exit(1) 

OUTPUT_DIR = "api_docs"
if not os.path.exists(OUTPUT_DIR):
    os.makedirs(OUTPUT_DIR)

session = requests.Session()

def login():
    url = f"{BASE_URL}/ipa/apis/json/general/login"
    payload = {
        "username": USERNAME,
        "password": PASSWORD
    }
    print(f"Logging in to {url}...")
    try:
        response = session.post(url, json=payload, verify=False) # Skip SSL verify for now as per client.go behavior
        response.raise_for_status()
        data = response.json()
        if data.get("result") == "SUCCESS":
            print("Login Successful")
            return True
        else:
            print(f"Login Failed: {data}")
            return False
    except Exception as e:
        print(f"Login Error: {e}")
        return False

def save_response(name, data):
    filename = os.path.join(OUTPUT_DIR, f"{name}.json")
    with open(filename, 'w', encoding='utf-8') as f:
        json.dump(data, f, indent=4, ensure_ascii=False)
    print(f"Saved {name} response to {filename}")

def get_provisioning_parameter_list():
    url = f"{BASE_URL}/ipa/apis/json/provisioning/getProvisioningParameterList"
    payload = {
        "username": USERNAME,
        "password": PASSWORD
    }
    print(f"Fetching {url}...")
    try:
        response = session.post(url, json=payload, verify=False)
        save_response("getProvisioningParameterList", response.json())
        return response.json()
    except Exception as e:
        print(f"Error fetching parameters: {e}")
        return None

def get_provisioning_data():
    url = f"{BASE_URL}/ipa/apis/json/provisioning/getProvisioningData"
    payload = {
        "username": USERNAME,
        "password": PASSWORD,
        "start": 0,
        "limit": 10,
        "sortDirection": "DESC",
        "search": []
    }
    print(f"Fetching {url}...")
    try:
        response = session.post(url, json=payload, verify=False)
        save_response("getProvisioningData", response.json())
        return response.json()
    except Exception as e:
        print(f"Error fetching data: {e}")
        return None

def get_provisioning_job_list():
    url = f"{BASE_URL}/ipa/apis/json/provisioning/getProvisioningJobList"
    payload = {
        "username": USERNAME,
        "password": PASSWORD,
        "start": 0,
        "limit": 10
    }
    print(f"Fetching {url}...")
    try:
        response = session.post(url, json=payload, verify=False)
        save_response("getProvisioningJobList", response.json())
        return response.json()
    except Exception as e:
        print(f"Error fetching jobs: {e}")
        return None


def generate_swagger_spec():
    print("Generating Swagger/OpenAPI specification...")
    
    swagger = {
        "openapi": "3.0.0",
        "info": {
            "title": "EyesOnT Pelephone API",
            "description": "API documentation for EyesOnT Pelephone Integration",
            "version": "1.5.2"
        },
        "servers": [
            {
                "url": BASE_URL,
                "description": "Production Server"
            }
        ],
        "paths": {},
        "components": {
            "schemas": {
                "RequestBase": {
                    "type": "object",
                    "properties": {
                        "username": {"type": "string"},
                        "password": {"type": "string"}
                    },
                    "required": ["username", "password"]
                },
                "ResponseBase": {
                    "type": "object",
                    "properties": {
                        "result": {"type": "string", "enum": ["SUCCESS", "REJECTED", "INVALID_REQ", "MISSING_ENTITY", "FAILED"]},
                        "message": {"type": "string", "nullable": True}
                    }
                }
            }
        }
    }

    # Helper to load response
    def load_json(name):
        try:
            with open(os.path.join(OUTPUT_DIR, f"{name}.json"), 'r', encoding='utf-8') as f:
                return json.load(f)
        except FileNotFoundError:
            return {}

    # 1. Login
    swagger["paths"]["/ipa/apis/json/general/login"] = {
        "post": {
            "summary": "Login to the system",
            "operationId": "login",
            "requestBody": {
                "required": True,
                "content": {
                    "application/json": {
                        "schema": {
                            "allOf": [
                                {"$ref": "#/components/schemas/RequestBase"}
                            ]
                        }
                    }
                }
            },
            "responses": {
                "200": {
                    "description": "Successful login",
                    "content": {
                        "application/json": {
                            "schema": {
                                "type": "object",
                                "properties": {
                                    "result": {"type": "string"},
                                    "sessionId": {"type": "string"},
                                    "jwtToken": {"type": "string"}
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    # 2. Logout
    swagger["paths"]["/ipa/apis/json/general/logout"] = {
        "post": {
            "summary": "Logout from the system",
            "operationId": "logout",
            "requestBody": {
                 "required": True,
                "content": {
                    "application/json": {
                        "schema": {
                             "allOf": [
                                {"$ref": "#/components/schemas/RequestBase"}
                            ]
                        }
                    }
                }
            },
            "responses": {
                "200": {
                    "description": "Logout response",
                    "content": {
                        "application/json": {
                            "schema": {
                                "$ref": "#/components/schemas/ResponseBase"
                            }
                        }
                    }
                }
            }
        }
    }

    # 3. GetProvisioningParameterList
    params_resp = load_json("getProvisioningParameterList")
    
    # Build dynamic schema for Subscriber from parameters
    subscriber_properties = {}
    
    if params_resp.get("parameters"):
        for param in params_resp["parameters"]:
            field_name = param.get("fieldName")
            if field_name:
                # Default to string
                prop_schema = {"type": "string", "nullable": True, "description": param.get("alias", "")}
                
                # Check for enums
                if param.get("availableValues"):
                     enums = [v["name"] for v in param["availableValues"]]
                     if enums:
                         prop_schema["enum"] = enums
                
                subscriber_properties[field_name] = prop_schema

    # Fallback/Additions based on data observation
    data_resp = load_json("getProvisioningData")
    if data_resp.get("data") and len(data_resp["data"]) > 0:
        for key, val in data_resp["data"][0].items():
            if key not in subscriber_properties:
                 if isinstance(val, int):
                    subscriber_properties[key] = {"type": "integer"}
                 else:
                    subscriber_properties[key] = {"type": "string", "nullable": True}

    swagger["paths"]["/ipa/apis/json/provisioning/getProvisioningParameterList"] = {
        "post": {
            "summary": "Get list of provisioning parameters",
            "operationId": "getProvisioningParameterList",
            "requestBody": {
                "required": True,
                "content": {
                    "application/json": {
                        "schema": {
                            "$ref": "#/components/schemas/RequestBase"
                        }
                    }
                }
            },
            "responses": {
                "200": {
                    "description": "List of parameters",
                    "content": {
                        "application/json": {
                            "schema": {
                                "allOf": [
                                    {"$ref": "#/components/schemas/ResponseBase"},
                                    {
                                        "type": "object",
                                        "properties": {
                                            "parameters": {
                                                "type": "array",
                                                "items": {
                                                    "type": "object",
                                                    "properties": {
                                                        "fieldName": {"type": "string"},
                                                        "permissionLevel": {"type": "string"},
                                                        "alias": {"type": "string"},
                                                        "availableValues": {
                                                            "type": "array",
                                                            "items": {
                                                                "type": "object",
                                                                "properties": {
                                                                    "value": {"type": "integer"},
                                                                    "name": {"type": "string"},
                                                                    "desc": {"type": "string"}
                                                                }
                                                            }
                                                        }
                                                    }
                                                }
                                            }
                                        }
                                    }
                                ]
                            }
                        }
                    }
                }
            }
        }
    }

    # 4. GetProvisioningData
    # Use captured data to verify fields
    data_resp = load_json("getProvisioningData")
    example_item = {}
    if data_resp.get("data") and len(data_resp["data"]) > 0:
        example_item = data_resp["data"][0]

    # Dynamically build the 'data' item schema based on the example
    data_item_properties = {}
    for k, v in example_item.items():
        if isinstance(v, int):
            data_item_properties[k] = {"type": "integer"}
        elif isinstance(v, float):
             data_item_properties[k] = {"type": "number"}
        else:
            data_item_properties[k] = {"type": "string", "nullable": True}

    swagger["paths"]["/ipa/apis/json/provisioning/getProvisioningData"] = {
        "post": {
            "summary": "Get provisioning data (SIMs)",
            "operationId": "getProvisioningData",
            "requestBody": {
                "required": True,
                "content": {
                    "application/json": {
                        "schema": {
                            "allOf": [
                                {"$ref": "#/components/schemas/RequestBase"},
                                {
                                    "type": "object",
                                    "properties": {
                                        "start": {"type": "integer", "default": 0},
                                        "limit": {"type": "integer", "default": 50},
                                        "sortBy": {"type": "string"},
                                        "sortDirection": {"type": "string", "enum": ["ASC", "DESC"]},
                                        "search": {
                                            "type": "array",
                                            "items": {
                                                "type": "object",
                                                "properties": {
                                                    "fieldName": {"type": "string"},
                                                    "fieldValue": {"type": "string"}
                                                }
                                            }
                                        }
                                    }
                                }
                            ]
                        }
                    }
                }
            },
            "responses": {
                "200": {
                    "description": "List of subscriber data",
                    "content": {
                        "application/json": {
                            "schema": {
                                "allOf": [
                                    {"$ref": "#/components/schemas/ResponseBase"},
                                    {
                                        "type": "object",
                                        "properties": {
                                            "count": {"type": "integer"},
                                            "fieldNames": {
                                                "type": "array",
                                                "items": {"type": "string"}
                                            },
                                            "data": {
                                                "type": "array",
                                                "items": {
                                                    "type": "object",
                                                    "properties": subscriber_properties
                                                }
                                            }
                                        }
                                    }
                                ]
                            }
                        }
                    }
                }
            }
        }
    }

    # 5. GetProvisioningJobList
    swagger["paths"]["/ipa/apis/json/provisioning/getProvisioningJobList"] = {
        "post": {
             "summary": "Get provisioning jobs",
            "operationId": "getProvisioningJobList",
            "requestBody": {
                "required": True,
                "content": {
                    "application/json": {
                        "schema": {
                             "allOf": [
                                {"$ref": "#/components/schemas/RequestBase"},
                                {
                                    "type": "object",
                                    "properties": {
                                        "start": {"type": "integer"},
                                        "limit": {"type": "integer"},
                                        "jobId": {"type": "integer"},
                                        "jobStatus": {"type": "string"}
                                    }
                                }
                            ]
                        }
                    }
                }
            },
            "responses": {
                "200": {
                    "description": "List of jobs",
                    "content": {
                        "application/json": {
                            "schema": {
                                "allOf": [
                                    {"$ref": "#/components/schemas/ResponseBase"},
                                    {
                                        "type": "object",
                                        "properties": {
                                            "count": {"type": "integer"},
                                            "jobs": {
                                                "type": "array",
                                                "items": {
                                                    "type": "object",
                                                    "properties": {
                                                        "jobId": {"type": "integer"},
                                                        "status": {"type": "string"},
                                                        "requestTime": {"type": "number"},
                                                        "actions": {
                                                            "type": "array",
                                                            "items": {
                                                                "type": "object",
                                                                "properties": {
                                                                    "neId": {"type": "string"},
                                                                    "status": {"type": "string"},
                                                                    "actionType": {"type": "string"},
                                                                    "targetValue": {"type": "string"}
                                                                }
                                                            }
                                                        }
                                                    }
                                                }
                                            }
                                        }
                                    }
                                ]
                            }
                        }
                    }
                }
            }
        }
    }

    # 6. UpdateProvisioningData
    swagger["paths"]["/ipa/apis/json/provisioning/updateProvisioningData"] = {
        "post": {
            "summary": "Update subscriber provisioning",
            "operationId": "updateProvisioningData",
             "requestBody": {
                "required": True,
                "content": {
                    "application/json": {
                        "schema": {
                             "allOf": [
                                {"$ref": "#/components/schemas/RequestBase"},
                                {
                                    "type": "object",
                                    "properties": {
                                        "actions": {
                                            "type": "array",
                                            "items": {
                                                "type": "object",
                                                "required": ["actionType", "targetValue", "subscribers"],
                                                "properties": {
                                                    "actionType": {"type": "string", "description": "e.g. SIM_STATE_CHANGE, RATE_PLAN_CHANGE"},
                                                    "targetValue": {"type": "string"},
                                                    "targetId": {"type": "string"},
                                                    "subscribers": {
                                                        "type": "array",
                                                        "items": {
                                                            "type": "object",
                                                            "properties": {
                                                                "neId": {"type": "string", "description": "MSISDN/CLI"}
                                                            }
                                                        }
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                            ]
                        }
                    }
                }
            },
            "responses": {
                "200": {
                    "description": "Update result",
                    "content": {
                        "application/json": {
                            "schema": {
                                "allOf": [
                                    {"$ref": "#/components/schemas/ResponseBase"},
                                    {
                                        "type": "object",
                                        "properties": {
                                            "requestId": {"type": "integer"}
                                        }
                                    }
                                ]
                            }
                        }
                    }
                }
            }
        }
    }

    output_file = os.path.join(OUTPUT_DIR, "swagger.json")
    with open(output_file, 'w', encoding='utf-8') as f:
        json.dump(swagger, f, indent=2)
    print(f"Swagger specification generated at {output_file}")

if __name__ == "__main__":
    import urllib3
    urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
    
    # Check if files exist, if not, fetch
    if not os.path.exists(os.path.join(OUTPUT_DIR, "getProvisioningData.json")):
        if login():
            get_provisioning_parameter_list()
            get_provisioning_data()
            get_provisioning_job_list()
    else:
        print("Using cached API responses...")
    
    # Always generate Swagger
    generate_swagger_spec()


