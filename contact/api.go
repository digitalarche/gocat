package contact

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/mitre/gocat/output"
)

var (
	apiBeacon = "/beacon"
)

//API communicates through HTTP
type API struct {
	name string
	client *http.Client
}

func init() {
	CommunicationChannels["HTTP"] = &API{ name: "HTTP" }
}

//GetInstructions sends a beacon and returns response.
func (a *API) GetBeaconBytes(profile map[string]interface{}) []byte {
	data, err := json.Marshal(profile)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot request beacon. Error with profile marshal: %s", err.Error()))
		return nil
	} else {
		address := fmt.Sprintf("%s%s", profile["server"], apiBeacon)
		return a.request(address, data)
	}
}

// Return the file bytes for the requested payload.
func (a *API) GetPayloadBytes(profile map[string]interface{}, payload string) ([]byte, string) {
    var payloadBytes []byte
    var filename string
    server := profile["server"]
    platform := profile["platform"]
    if server != nil && platform != nil {
		address := fmt.Sprintf("%s/file/download", server.(string))
		req, err := http.NewRequest("POST", address, nil)
		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[-] Failed to create HTTP request: %s", err.Error()))
			return nil, ""
		}
		req.Header.Set("file", payload)
		req.Header.Set("platform", platform.(string))
		req.Header.Set("paw", profile["paw"].(string))
		resp, err := a.client.Do(req)
		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[-] Error sending payload request: %s", err.Error()))
			return nil, ""
		}
		defer resp.Body.Close()
		if resp.StatusCode == ok {
			buf, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				output.VerbosePrint(fmt.Sprintf("[-] Error reading HTTP response: %s", err.Error()))
				return nil, ""
			}
			payloadBytes = buf
			if name_header, ok := resp.Header["Filename"]; ok {
				filename = filepath.Join(name_header[0])
			} else {
				output.VerbosePrint("[-] HTTP response missing Filename header.")
			}
		}
    }
	return payloadBytes, filename
}

//C2RequirementsMet determines if sandcat can use the selected comm channel
func (a *API) C2RequirementsMet(profile map[string]interface{}, c2Config map[string]string) (bool, map[string]string) {
	output.VerbosePrint(fmt.Sprintf("Beacon API=%s", apiBeacon))
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// Handle proxy gateway configuration.
	if proxyUrlStr, ok := c2Config["httpProxyGateway"]; ok && len(proxyUrlStr) > 0 {
		proxyUrl, err := url.Parse(proxyUrlStr)
		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[!] Error - could not establish HTTP proxy requirements: %s", err.Error()))
			return false, nil
		}
		http.DefaultTransport.(*http.Transport).Proxy = http.ProxyURL(proxyUrl)
	}
	a.client = &http.Client{Transport: http.DefaultTransport}

	return true, nil
}

//SendExecutionResults will send the execution results to the server.
func (a *API) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}) {
	address := fmt.Sprintf("%s%s", profile["server"], apiBeacon)
	profileCopy := make(map[string]interface{})
	for k,v := range profile {
		profileCopy[k] = v
	}
	results := make([]map[string]interface{}, 1)
	results[0] = result
	profileCopy["results"] = results
	data, err := json.Marshal(profileCopy)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot send results. Error with profile marshal: %s", err.Error()))
	} else {
		a.request(address, data)
	}
}

func (a *API) GetName() string {
	return a.name
}

func (a *API) request(address string, data []byte) []byte {
	encodedData := []byte(base64.StdEncoding.EncodeToString(data))
	req, err := http.NewRequest("POST", address, bytes.NewBuffer(encodedData))
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Failed to create HTTP request: %s", err.Error()))
		return nil
	}
	resp, err := a.client.Do(req)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Failed to perform HTTP request: %s", err.Error()))
		return nil
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Failed to read HTTP response: %s", err.Error()))
		return nil
	}
	decodedBody, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Failed to decode HTTP response: %s", err.Error()))
		return nil
	}
	return decodedBody
}