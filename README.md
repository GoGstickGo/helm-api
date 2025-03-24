# API Documentation

## Overview
This API provides endpoints for managing environments through Helm. All endpoints support CORS with the following configuration:
- Allowed Origins: `*`
- Allowed Methods: `GET`, `POST`
- Allowed Headers: `Accept`, `Content-Type`

## Authentication
Environment management endpoints (create, update, delete) require an API key in the request header:
```
X-API-Key: your-api-key

```

## Endpoints

### Create Environment
Creates a new environment using Helm.

**Endpoint**: `POST /create-env`  
**Authentication**: Required (API key)

**Request Headers**:
```
Content-Type: application/json
X-API-Key: your-api-key
```

**Request Body**:
```json
{
  "name": "chart1",
  "version": "0.1.0", 
  "description": "A custom Helm chart",
	"apiversion": "v2",
	"type": "application"
}
```

**Response**:
* 201: Environment created successfully
* 400: Invalid request body
* 401: Unauthorized (invalid API key)
* 500: Internal server error

### Update Environment
Updates an existing environment. 

**Endpoint**: `POST /update-env/{chartName}`  
**Authentication**: Required (API key)

**URL Parameters**:
* chartName: Name of the chart environment to update

**Request Headers**:
```
Content-Type: application/json
X-API-Key: your-api-key
```

**Request Body**:
```json
{
    "action: "up | down",
}
```

**Response**:
* 200: Environment updated successfully
* 400: Invalid request body
* 401: Unauthorized (invalid API key)
* 500: Internal server error

### Delete Environment
Deletes an existing environment.

**Endpoint**: `POST /delete-env/{chartName}`  
**Authentication**: Required (API key)

**URL Parameters**:
* chartName: Name of the chart environment to delete

**Request Headers**:
```
X-API-Key: your-api-key
```

**Response**:
* 200: Environment deleted successfully
* 401: Unauthorized (invalid API key)
* 500: Internal server error

### List Environments
Lists all available environments.

**Endpoint**: `GET /list`  
**Authentication**: Not required

**Response**:
* 200: List retrieved successfully
```json
{
    "data": [
        {

        }
    ]
}
```
* 500: Internal server error

### Health Check
Checks the API service health status.

**Endpoint**: `GET /health-check`  
**Authentication**: Not required

**Response**:
* 200: Service is healthy
```json
{
    "message":"API is healthy"
}
```
* 503: Service is unhealthy

## Error Responses
All endpoints may return these common error responses:

```json
{
    "error": "string",
    "message": "string"
}
```