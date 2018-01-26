package spec

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mackerelio/golib/logging"
	"github.com/monosense-products/mackerel-agent/config"
)

// This Generator collects metadata about cloud instances.
// Currently EC2 and GCE are supported.
// EC2: http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AESDG-chapter-instancedata.html
// GCE: https://developers.google.com/compute/docs/metadata
// AzureVM: https://docs.microsoft.com/azure/virtual-machines/virtual-machines-instancemetadataservice-overview

// CloudGenerator definition
type CloudGenerator struct {
	CloudMetaGenerator
}

// CloudMetaGenerator interface of metadata generator for each cloud platform
type CloudMetaGenerator interface {
	Generate() (interface{}, error)
	SuggestCustomIdentifier() (string, error)
}

// Key is a root key for the generator.
func (g *CloudGenerator) Key() string {
	return "cloud"
}

var cloudLogger = logging.GetLogger("spec.cloud")

var ec2BaseURL, gceMetaURL, azureVMBaseURL *url.URL

func init() {
	ec2BaseURL, _ = url.Parse("http://169.254.169.254/latest/meta-data")
	gceMetaURL, _ = url.Parse("http://metadata.google.internal/computeMetadata/v1/?recursive=true")
	azureVMBaseURL, _ = url.Parse("http://169.254.169.254/metadata/instance")
}

var timeout = 100 * time.Millisecond

// SuggestCloudGenerator returns suitable CloudGenerator
func SuggestCloudGenerator(conf *config.Config) *CloudGenerator {
	// if CloudPlatform is specified, return corresponding one
	switch conf.CloudPlatform {
	case config.CloudPlatformNone:
		return nil
	case config.CloudPlatformEC2:
		return &CloudGenerator{&EC2Generator{ec2BaseURL}}
	case config.CloudPlatformGCE:
		return &CloudGenerator{&GCEGenerator{gceMetaURL}}
	case config.CloudPlatformAzureVM:
		return &CloudGenerator{&AzureVMGenerator{azureVMBaseURL}}
	}

	if isEC2() {
		return &CloudGenerator{&EC2Generator{ec2BaseURL}}
	}
	if isGCE() {
		return &CloudGenerator{&GCEGenerator{gceMetaURL}}
	}
	if isAzure() {
		return &CloudGenerator{&AzureVMGenerator{azureVMBaseURL}}
	}

	return nil
}

func httpCli() *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			// don't use HTTP_PROXY when requesting cloud instance metadata APIs
			Proxy: nil,
		},
	}
}

func isGCE() bool {
	_, err := requestGCEMeta()
	return err == nil
}

// Note: May want to check without using the API.
func isAzure() bool {
	cl := httpCli()
	// '/vmId` is probably Azure VM specific URL
	req, err := http.NewRequest("GET", azureVMBaseURL.String()+"/compute/vmId?api-version=2017-04-02&format=text", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Metadata", "true")

	resp, err := cl.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func requestGCEMeta() ([]byte, error) {
	cl := httpCli()
	req, err := http.NewRequest("GET", gceMetaURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := cl.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to request gce meta. response code: %d", resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

// EC2Generator meta generator for EC2
type EC2Generator struct {
	baseURL *url.URL
}

// Generate collects metadata from cloud platform.
func (g *EC2Generator) Generate() (interface{}, error) {
	cl := httpCli()

	metadataKeys := []string{
		"instance-id",
		"instance-type",
		"placement/availability-zone",
		"security-groups",
		"ami-id",
		"hostname",
		"local-hostname",
		"public-hostname",
		"local-ipv4",
		"public-ipv4",
		"reservation-id",
	}

	metadata := make(map[string]string)

	for _, key := range metadataKeys {
		resp, err := cl.Get(g.baseURL.String() + "/" + key)
		if err != nil {
			cloudLogger.Debugf("This host may not be running on EC2. Error while reading '%s'", key)
			return nil, nil
		}
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				cloudLogger.Errorf("Results of requesting metadata cannot be read: '%s'", err)
				break
			}
			metadata[key] = string(body)
			cloudLogger.Debugf("results %s:%s", key, string(body))
		} else {
			cloudLogger.Debugf("Status code of the result of requesting metadata '%s' is '%d'", key, resp.StatusCode)
		}
	}

	results := make(map[string]interface{})
	results["provider"] = "ec2"
	results["metadata"] = metadata

	return results, nil
}

// SuggestCustomIdentifier suggests the identifier of the EC2 instance
func (g *EC2Generator) SuggestCustomIdentifier() (string, error) {
	cl := httpCli()
	key := "instance-id"
	resp, err := cl.Get(g.baseURL.String() + "/" + key)
	if err != nil {
		return "", fmt.Errorf("error while retrieving instance-id")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to request instance-id. response code: %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("results of requesting instance-id cannot be read: '%s'", err)
	}
	instanceID := string(body)
	if instanceID == "" {
		return "", fmt.Errorf("invalid instance id")
	}
	return instanceID + ".ec2.amazonaws.com", nil
}

// GCEGenerator generate for GCE
type GCEGenerator struct {
	metaURL *url.URL
}

// Generate collects metadata from cloud platform.
func (g *GCEGenerator) Generate() (interface{}, error) {
	bytes, err := requestGCEMeta()
	if err != nil {
		return nil, err
	}
	var data gceMeta
	json.Unmarshal(bytes, &data)
	return data.toGeneratorResults(), nil
}

type gceInstance struct {
	Zone         string
	InstanceType string `json:"machineType"`
	Hostname     string
	InstanceID   uint64 `json:"id"`
}

type gceProject struct {
	ProjectID        string
	NumericProjectID uint64
}

type gceMeta struct {
	Instance *gceInstance
	Project  *gceProject
}

func (g gceMeta) toGeneratorMeta() map[string]string {
	meta := make(map[string]string)

	lastS := func(s string) string {
		ss := strings.Split(s, "/")
		return ss[len(ss)-1]
	}

	if ins := g.Instance; ins != nil {
		meta["hostname"] = ins.Hostname
		meta["instance-id"] = fmt.Sprint(ins.InstanceID)
		meta["instance-type"] = lastS(ins.InstanceType)
		meta["zone"] = lastS(ins.Zone)
	}

	if proj := g.Project; proj != nil {
		meta["projectId"] = proj.ProjectID
	}

	return meta
}

func (g gceMeta) toGeneratorResults() interface{} {
	results := make(map[string]interface{})
	results["provider"] = "gce"
	results["metadata"] = g.toGeneratorMeta()

	return results
}

// SuggestCustomIdentifier for GCE is not implemented yet
func (g *GCEGenerator) SuggestCustomIdentifier() (string, error) {
	return "", nil
}

// AzureVMGenerator meta generator for Azure VM
type AzureVMGenerator struct {
	baseURL *url.URL
}

// Generate collects metadata from cloud platform.
func (g *AzureVMGenerator) Generate() (interface{}, error) {
	metadataComputeKeys := map[string]string{
		"location":  "location",
		"offer":     "imageReferenceOffer",
		"osType":    "osSystemType",
		"publisher": "imageReferencePublisher",
		"sku":       "imageReferenceSku",
		"vmId":      "vmID",
		"vmSize":    "virtualMachineSizeType",
	}

	ipAddressKeys := map[string]string{
		"privateIpAddress": "privateIpAddress",
		"publicIpAddress":  "publicIpAddress",
	}

	metadata := make(map[string]string)
	metadata = retrieveAzureVMMetadata(metadata, g.baseURL.String(), "/compute/", metadataComputeKeys)
	metadata = retrieveAzureVMMetadata(metadata, g.baseURL.String(), "/network/interface/0/ipv4/ipAddress/0/", ipAddressKeys)

	results := make(map[string]interface{})
	results["provider"] = "AzureVM"
	results["metadata"] = metadata

	return results, nil
}

func retrieveAzureVMMetadata(metadataMap map[string]string, baseURL string, urlSuffix string, keys map[string]string) map[string]string {
	cl := httpCli()

	for key, value := range keys {
		req, err := http.NewRequest("GET", baseURL+urlSuffix+key+"?api-version=2017-04-02&format=text", nil)
		if err != nil {
			cloudLogger.Debugf("This host may not be running on Azure VM. Error while reading '%s'", key)
			return metadataMap
		}

		req.Header.Set("Metadata", "true")

		resp, err := cl.Do(req)
		if err != nil {
			cloudLogger.Debugf("This host may not be running on Azure VM. Error while reading '%s'", key)
			return metadataMap
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				cloudLogger.Errorf("Results of requesting metadata cannot be read: '%s'", err)
				break
			}
			metadataMap[value] = string(body)
			cloudLogger.Debugf("results %s:%s", key, string(body))
		} else {
			cloudLogger.Debugf("Status code of the result of requesting metadata '%s' is '%d'", key, resp.StatusCode)
		}
	}
	return metadataMap
}

// SuggestCustomIdentifier suggests the identifier of the Azure VM instance
func (g *AzureVMGenerator) SuggestCustomIdentifier() (string, error) {
	cl := httpCli()
	req, err := http.NewRequest("GET", azureVMBaseURL.String()+"/compute/vmId?api-version=2017-04-02&format=text", nil)
	if err != nil {
		return "", fmt.Errorf("error while retrieving vmId")
	}
	req.Header.Set("Metadata", "true")

	resp, err := cl.Do(req)
	if err != nil {
		return "", fmt.Errorf("error while retrieving vmId")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to request vmId. response code: %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("results of requesting vmId cannot be read: '%s'", err)
	}
	instanceID := string(body)
	if instanceID == "" {
		return "", fmt.Errorf("invalid instance id")
	}
	return instanceID + ".virtual_machine.azure.microsoft.com", nil
}
