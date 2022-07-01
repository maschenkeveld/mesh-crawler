from flask import Flask, request, jsonify
#from elasticapm.contrib.flask import ElasticAPM
import requests
import yaml
import sys
import os
import json

app = Flask(__name__)

#               app.config['ELASTIC_APM'] = {
#                   # Set the required service name. Allowed characters:
#                   # a-z, A-Z, 0-9, -, _, and space
#                   'SERVICE_NAME': 'empo',
#                   # Use if APM Server requires a secret token
#                   'SECRET_TOKEN': '',
#                   # Set the custom APM Server URL (default: http://localhost:8200)
#                   'SERVER_URL': 'http://elk.mschnkvld.lab:8200',
#                   # Set the service environment
#                   'ENVIRONMENT': 'production',
#               }
#               apm = ElasticAPM(app)

@app.route('/', methods=['POST'])
def process_post():
    print("My name is {} and I RECEIVED a POST request from {}".format(myName, str(request.headers.get('Host'))))

    if (request.headers.get('Content-Type') == 'application/x-yaml'):   
        try:
            requestBody = yaml.safe_load(request.get_data())
            myNextHops = requestBody['nextHops']
            #mprint(myNextHops)
        except yaml.YAMLError as e:
            print(e)
        else:
            myResponse = {}
            myResponse["myName"] = str(myName)
            upstreamResponses = []

            for currentService in myNextHops:
                headers = {
                    "Content-Type": "application/x-yaml", 
                    "Mesh-Crawler-Requester": myName,
                    "x-request-id": request.headers.get("x-request-id"),
                    "x-b3-traceid": request.headers.get("x-b3-traceid"),
                    "x-b3-parentspanid": request.headers.get("x-b3-parentspanid"),
                    "x-b3-spanid": request.headers.get("x-b3-spanid"),
                    "x-b3-sampled": request.headers.get("x-b3-sampled"),
                    "x-b3-flags": request.headers.get("x-b3-flags")
                }

                currentUpstreamResponse = requests.Response()
                currentUpstreamExceptionResponse = {}

                try:
                    if "nextHops" in currentService:
                        currentUpstream = str(currentService['serviceUrl'])
                        print("My name is {} and I will SEND a POST request to {}".format(myName, str(currentUpstream)))
                        currentUpstreamMethod = "POST"
                        currentUpstreamRequest = {'nextHops': currentService['nextHops']}
                        currentUpstreamResponse = requests.post(url = currentUpstream, data = str(yaml.dump(currentUpstreamRequest)), headers = headers)
                    else:
                        currentUpstream = str(currentService['serviceUrl'])
                        print("My name is {} and I will SEND a GET request to {}".format(myName, str(currentUpstream)))
                        currentUpstreamMethod = "GET"
                        currentUpstreamResponse = requests.get(url = currentUpstream, headers = headers)

                except requests.exceptions.InvalidJSONError:
                    print("A JSON error occurred.") # APPEND THIS TO MYUPSTREAMRESPONSES DICT!!! AS ERROR
                    currentUpstreamExceptionResponse["Exception"] = "A JSON error occurred." # APPEND THIS TO MYUPSTREAMRESPONSES DICT!!! AS ERROR
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except requests.exceptions.HTTPError as e:
                    print("An HTTP error occurred.")
                    currentUpstreamExceptionResponse["Exception"] = "An HTTP error occurred."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except requests.exceptions.ConnectionError as e:
                    print("A Connection error occurred.")
                    currentUpstreamExceptionResponse["Exception"] = "A Connection error occurred."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                # except requests.exceptions.ProxyError as e:
                #     print("A proxy error occurred.")
                #     currentUpstreamExceptionResponse["Exception"] = "A proxy error occurred."
                #     currentUpstreamExceptionResponse["ErrorString"] = str(e)
                # except requests.exceptions.SSLError as e:
                #     print("An SSL error occurred.")
                #     currentUpstreamExceptionResponse["Exception"] = "An SSL error occurred."
                #     currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except requests.exceptions.Timeout as e:
                    print("The request timed out.")
                    currentUpstreamExceptionResponse["Exception"] = "The request timed out."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                # except requests.exceptions.ConnectTimeout as e:
                #     print("The request timed out while trying to connect to the remote server.")
                #     currentUpstreamExceptionResponse["Exception"] = "The request timed out while trying to connect to the remote server."
                #     currentUpstreamExceptionResponse["ErrorString"] = str(e)
                # except requests.exceptions.ReadTimeout as e:
                #     print("The server did not send any data in the allotted amount of time.")
                #     currentUpstreamExceptionResponse["Exception"] = "The server did not send any data in the allotted amount of time."
                #     currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except requests.exceptions.URLRequired as e:
                    print("A valid URL is required to make a request.")
                    currentUpstreamExceptionResponse["Exception"] = "A valid URL is required to make a request."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except requests.exceptions.TooManyRedirects as e:
                    print("Too many redirects.")
                    currentUpstreamExceptionResponse["Exception"] = "Too many redirects."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except requests.exceptions.MissingSchema as e:
                    print("The URL schema (e.g. http or https) is missing.")
                    currentUpstreamExceptionResponse["Exception"] = "The URL schema (e.g. http or https) is missing."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except requests.exceptions.InvalidSchema as e:
                    print("See defaults.py for valid schemas.")
                    currentUpstreamExceptionResponse["Exception"] = "See defaults.py for valid schemas."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except requests.exceptions.InvalidURL as e:
                    print("The URL provided was somehow invalid.")
                    currentUpstreamExceptionResponse["Exception"] = "The URL provided was somehow invalid."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except requests.exceptions.InvalidHeader as e:
                    print("The header value provided was somehow invalid.")
                    currentUpstreamExceptionResponse["Exception"] = "The header value provided was somehow invalid."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                # except requests.exceptions.InvalidProxyURL as e:
                #     print("The proxy URL provided is invalid.")
                #     currentUpstreamExceptionResponse["Exception"] = "The proxy URL provided is invalid."
                #     currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except requests.exceptions.ChunkedEncodingError as e:
                    print("The server declared chunked encoding but sent an invalid chunk.")
                    currentUpstreamExceptionResponse["Exception"] = "The server declared chunked encoding but sent an invalid chunk."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except requests.exceptions.ContentDecodingError as e:
                    print("Failed to decode response content.")
                    currentUpstreamExceptionResponse["Exception"] = "Failed to decode response content."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except requests.exceptions.StreamConsumedError as e:
                    print("The content for this response was already consumed.")
                    currentUpstreamExceptionResponse["Exception"] = "The content for this response was already consumed."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except requests.exceptions.RetryError as e:
                    print("Custom retries logic failed.")
                    currentUpstreamExceptionResponse["Exception"] = "Custom retries logic failed."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except requests.exceptions.UnrewindableBodyError as e:
                    print("Requests encountered an error when trying to rewind a body.")
                    currentUpstreamExceptionResponse["Exception"] = "Requests encountered an error when trying to rewind a body."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except requests.exceptions.RequestsWarning as e:
                    print("Base warning for Requests.")
                    currentUpstreamExceptionResponse["Exception"] = "Base warning for Requests."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                # except requests.exceptions.FileModeWarning as e:
                #     print("A file was opened in text mode, but Requests determined its binary length.")
                #     currentUpstreamExceptionResponse["Exception"] = "A file was opened in text mode, but Requests determined its binary length."
                #     currentUpstreamExceptionResponse["ErrorString"] = str(e)
                # except requests.exceptions.RequestsDependencyWarning as e:
                #     print("An imported dependency doesn't match the expected version range.")
                #     currentUpstreamExceptionResponse["Exception"] = "An imported dependency doesn't match the expected version range."
                #     currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except requests.exceptions.RequestException as e:
                    print("A Request Exceptions occured.")
                    currentUpstreamExceptionResponse["Exception"] = "A Request Exceptions occured."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                except Exception as e:
                    print(e)
                    currentUpstreamExceptionResponse["Exception"] = "We are not looking good."
                    currentUpstreamExceptionResponse["ErrorString"] = str(e)
                else:
                    if(len(currentUpstreamResponse.content) == 0):
                        currentUpstreamExceptionResponse["Exception"] = "Received an empty response from upstream."
                finally:
                    currentUpstreamResponseDict = {}

                    if(len(currentUpstreamExceptionResponse) > 0):
                        currentUpstreamResponseDict["[{}] - {}".format(str(currentUpstreamMethod), str(currentUpstream))] = currentUpstreamExceptionResponse
                    else:
                        currentUpstreamResponseDict["[{}] - {}".format(str(currentUpstreamMethod), str(currentUpstream))] = currentUpstreamResponse.json()

                    upstreamResponses.append(currentUpstreamResponseDict)

            myResponse["myIncomingRequestheaders"] = headersToHeadersDict(request.headers)
            myResponse["myUpstreamResponses"] = upstreamResponses

    return myResponse

@app.route('/', methods=['GET'])
def process_get():
    print("My name is {} and I RECEIVED a GET request from {}".format(myName, str(request.headers.get('Host'))))

    myResponse = {}
    myResponse["myName"] = str(myName)
    myResponse["myIncomingRequestheaders"] = headersToHeadersDict(request.headers)

    return myResponse

def headersToHeadersDict(headers):
    headersDict = {}
    for header in headers:
        headersDict[header[0]] = header[1]
    return headersDict

if __name__ == "__main__":
    global myName
    myName = str(os.getenv('app_name'))
    appPort = str(os.getenv('app_port'))
    print("My name is {} and I will listen on port {}".format(myName, appPort))
    #app.run(host='0.0.0.0', port=appPort, ssl_context="adhoc")
    app.run(host='0.0.0.0', port=appPort)