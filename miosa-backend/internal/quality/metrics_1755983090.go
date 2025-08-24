import time
from prometheus_client import start_http_server, Summary

# Create a metric to track the authentication latency
AUTH_LATENCY = Summary('auth_latency_seconds', 'Authentication latency in seconds')

# Create a metric to track the number of successful authentications
SUCCESSFUL_AUTH = Summary('successful_auths', 'Number of successful authentications')

# Create a metric to track the number of failed authentications
FAILED_AUTH = Summary('failed_auths', 'Number of failed authentications')

# Start the Prometheus server
start_http_server(8000)