package openstack

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

var hatcheryOpenStack *HatcheryCloud

var mapWorkerBinaries = map[string][]byte{}

// HatcheryCloud spawns instances of worker model with type 'ISO'
// by startup up virtual machines on /cloud
type HatcheryCloud struct {
	hatch     *sdk.Hatchery
	token     *Token
	networkID string
	ips       []string
	flavors   []Flavor
	networks  []Network

	// User provided parameters
	address   string
	user      string
	password  string
	endpoint  string
	tenant    string
	region    string
	network   string
	workerTTL int
}

// ID returns hatchery id
func (h *HatcheryCloud) ID() int64 {
	if h.hatch == nil {
		return 0
	}
	return h.hatch.ID
}

//Hatchery returns hatchery instance
func (h *HatcheryCloud) Hatchery() *sdk.Hatchery {
	return h.hatch
}

// ModelType returns type of hatchery
func (*HatcheryCloud) ModelType() string {
	return sdk.Openstack
}

// CanSpawn return wether or not hatchery can spawn model
// requirements are not supported
func (h *HatcheryCloud) CanSpawn(model *sdk.Model, job *sdk.PipelineBuildJob) bool {
	for _, r := range job.Job.Action.Requirements {
		if r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			return false
		}
	}
	return true
}

const serverStatusBuild = "BUILD"
const serverStatusActive = "ACTIVE"

// Init fetch uri from nova
// then list available models
// then list available images
func (h *HatcheryCloud) Init() error {
	// Register without declaring model
	h.hatch = &sdk.Hatchery{
		Name: hatchery.GenerateName("openstack", viper.GetString("name")),
		UID:  viper.GetString("uk"),
	}

	if errRegistrer := hatchery.Register(h.hatch, viper.GetString("token")); errRegistrer != nil {
		log.Warning("Cannot register hatchery: %s", errRegistrer)
	}

	var errt error
	h.token, h.endpoint, errt = getToken(h.user, h.password, h.address, h.tenant, h.region)
	if errt != nil {
		return errt
	}

	h.refreshToken()
	log.Debug("NewOpenstackStore> Got token %dchar at %s", len(h.token.ID), h.endpoint)

	var errf error
	h.flavors, errf = getFlavors(h.endpoint, h.token.ID)
	if errf != nil {
		log.Warning("Error getting flavors: %s", errf)
	}
	var errn error
	h.networks, errn = getNetworks(h.endpoint, h.token.ID)
	if errn != nil {
		log.Warning("Error getting networks: %s", errn)
	}
	var errni error
	h.networkID, errni = h.getNetworkID(h.network)
	if errni != nil {
		return fmt.Errorf("cannot find network '%s'", h.network)
	}

	//Download the worker binary witch should be injected in servers
	//FIXME: only linux is supported for the moment. Windows worker binary can be downloaded but, he have to manager OS requirement first
	var code int
	var errd error
	mapWorkerBinaries["linux_x86_64"], code, errd = sdk.Request("GET", "/download/worker/x86_64", nil)
	if errd != nil || code != 200 {
		log.Fatalf("Unable to download worker binary from api. This is fatal...")
		os.Exit(10)
	}

	go h.main()

	return nil
}

func (h *HatcheryCloud) main() {
	serverListTick := time.NewTicker(10 * time.Second).C
	killAwolServersTick := time.NewTicker(30 * time.Second).C
	killErrorServersTick := time.NewTicker(60 * time.Second).C
	killDisabledWorkersTick := time.NewTicker(60 * time.Second).C
	refreshTokenTick := time.NewTicker(20 * time.Hour).C

	for {
		select {

		case <-serverListTick:
			h.updateServerList()

		case <-killAwolServersTick:
			h.killAwolServers()

		case <-killErrorServersTick:
			h.killErrorServers()

		case <-killDisabledWorkersTick:
			h.killDisabledWorkers()

		case <-refreshTokenTick:
			h.refreshToken()
		}
	}
}

func (h *HatcheryCloud) killDisabledWorkers() {
	workers, err := sdk.GetWorkers()
	if err != nil {
		log.Warning("killDisabledWorkers> Cannot fetch worker list: %s", err)
		return
	}
	servers := h.getServers()

	for _, w := range workers {
		if w.Status != sdk.StatusDisabled {
			continue
		}

		for _, s := range servers {
			if s.Name == w.Name {
				log.Notice("Deleting disabled worker %s", w.Name)
				err := deleteServer(h.endpoint, h.token.ID, s.ID)
				if err != nil {
					log.Warning("Cannot remove server %s: %s", s.Name, err)
					continue
				}
			}
		}
	}
}

func (h *HatcheryCloud) killErrorServers() {
	for _, s := range h.getServers() {
		//Remove server without IP Address
		if s.Status == "ACTIVE" {
			t, err := time.Parse(time.RFC3339, s.Updated)
			if err != nil {
				log.Warning("killErrorServers> Cannot parse last update: %s", err)
				continue
			}

			if len(s.Addresses) == 0 && time.Since(t) > 6*time.Minute {
				log.Notice("Deleting server %s without IP Address", s.Name)
				if err := deleteServer(h.endpoint, h.token.ID, s.ID); err != nil {
					log.Warning("Cannot remove server %s: %s", s.Name, err)
					continue
				}
			}
		}

		//Remove Error server
		if s.Status == "ERROR" {
			log.Notice("Deleting server %s in error", s.Name)
			if err := deleteServer(h.endpoint, h.token.ID, s.ID); err != nil {
				log.Warning("Cannot remove server %s: %s", s.Name, err)
				continue
			}
		}
	}
}

func (h *HatcheryCloud) killAwolServers() {
	var found bool

	workers, err := sdk.GetWorkers()
	if err != nil {
		log.Warning("killAwolServers> Cannot fetch worker list: %s", err)
		return
	}

	for _, s := range h.getServers() {
		log.Debug("killAwolServers> Checking %s (%s) %v", s.Name, s.ImageRef, s.Metadata)
		if s.Status == "BUILD" {
			continue
		}
		found = false
		for _, w := range workers {
			if w.Name == s.Name {
				found = true
				break
			}
		}

		if found {
			continue
		}

		workerHatcheryName, _ := s.Metadata["hatcheryName"]
		_, ok := s.Metadata["worker"]

		t, err := time.Parse(time.RFC3339, s.Updated)
		if err != nil {
			log.Warning("Cannot parse last update: %s", err)
			break
		}

		log.Debug("killAwolServers> Deleting %s (%s) %v ? : %v %v hatcheryName:%s %v", s.Name, s.ImageRef, s.Metadata, ok, (time.Since(t) > 6*time.Minute), workerHatcheryName, (workerHatcheryName == h.Hatchery().Name))

		// Delete workers, if not identified by CDS API
		// Wait for 6 minutes, to avoid killing worker babies
		if (workerHatcheryName == "" || workerHatcheryName == h.Hatchery().Name) && ok && time.Since(t) > 6*time.Minute {
			log.Notice("killAwolServers> %s last update: %s", s.Name, time.Since(t))
			if err := deleteServer(h.endpoint, h.token.ID, s.ID); err != nil {
				log.Warning("killAwolServers> Cannot remove server %s: %s", s.Name, err)
			}
		}
	}
}

func (h *HatcheryCloud) refreshToken() {
	tk, endpoint, err := getToken(h.user, h.password, h.address, h.tenant, h.region)
	if err != nil {
		log.Critical("refreshToken> Cannot refresh token: %s", err)
	}
	h.token = tk
	h.endpoint = endpoint
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryCloud) WorkersStarted() int {
	return len(h.getServers())
}

// WorkersStartedByModel returns the number of instances of given model started but
// not necessarily register on CDS yet
func (h *HatcheryCloud) WorkersStartedByModel(model *sdk.Model) int {
	var x int
	for _, s := range h.getServers() {
		if strings.Contains(s.Name, strings.ToLower(model.Name)) {
			x++
		}
	}
	log.Info("WorkersStartedByModel> %s : %d", model.Name, x)
	return x
}

// KillWorker delete cloud instances
func (h *HatcheryCloud) KillWorker(worker sdk.Worker) error {
	log.Notice("KillWorker> Kill %s", worker.Name)
	for _, s := range h.getServers() {
		if s.Name == worker.Name {
			if err := deleteServer(h.endpoint, h.token.ID, s.ID); err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("not found")
}

// SpawnWorker creates a new cloud instances
// requirements are not supported
func (h *HatcheryCloud) SpawnWorker(model *sdk.Model, job *sdk.PipelineBuildJob) error {
	if job != nil {
		log.Notice("spawnWorker> spawning worker %s for job %d", model.Name, job.ID)
	} else {
		log.Notice("spawnWorker> spawning worker %s ", model.Name)
	}

	var err error
	var omd sdk.OpenstackModelData

	if h.hatch == nil {
		return fmt.Errorf("hatchery disconnected from engine")
	}

	if len(h.getServers()) == viper.GetInt("max-worker") {
		log.Info("MaxWorker limit (%d) reached", viper.GetInt("max-worker"))
		return nil
	}

	if err = json.Unmarshal([]byte(model.Image), &omd); err != nil {
		return err
	}

	// Get image ID
	imageID, err := h.imageID(omd.Image)
	if err != nil {
		return err
	}

	// Get flavor ID
	flavorID, err := h.flavorID(omd.Flavor)
	if err != nil {
		return err
	}

	//FIXME => 413 entity too large
	/* Inject worker binary file
	personnality := []*File{
		&File{
			Path:     "/worker",
			Contents: mapWorkerBinaries["linux_x86_64"],
		},
	}*/
	personnality := []*File{}

	// Ip len(h.ips) > 0, specify one of those
	var ip string
	if len(h.ips) > 0 {
		ip, err = h.findAvailableIP()
		log.Debug("Found %s as first available IP", ip)
		if err != nil {
			return err
		}
	}

	//generate a pretty cool name
	name := model.Name + "-" + strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1)

	// Decode base64 given user data
	udataModel, err := base64.StdEncoding.DecodeString(omd.UserData)
	if err != nil {
		return err
	}

	// Add curl of worker
	udataBegin := `#!/bin/sh
set +e
`
	udataEnd := `
cd $HOME
# Download and start worker with curl
curl  "{{.API}}/download/worker/$(uname -m)" -o worker --retry 10 --retry-max-time 0 -C - >> /tmp/user_data 2>&1
chmod +x worker
CDS_SINGLE_USE=1 ./worker --api={{.API}} --key={{.Key}} --name={{.Name}} --model={{.Model}} --hatchery={{.Hatchery}} --booked-job-id={{.JobID}} --single-use --ttl={{.TTL}} && exit 0
`
	var udata = udataBegin + string(udataModel) + udataEnd

	var jobID int64
	if job != nil {
		jobID = job.ID
	}

	tmpl, errt := template.New("udata").Parse(string(udata))
	if errt != nil {
		return errt
	}
	udataParam := struct {
		API      string
		Name     string
		Key      string
		Model    int64
		Hatchery int64
		JobID    int64
		TTL      int
	}{
		API:      viper.GetString("api"),
		Name:     name,
		Key:      viper.GetString("token"),
		Model:    model.ID,
		Hatchery: h.hatch.ID,
		JobID:    jobID,
		TTL:      h.workerTTL,
	}
	var buffer bytes.Buffer
	if err = tmpl.Execute(&buffer, udataParam); err != nil {
		return err
	}

	// Encode again
	udata64 := base64.StdEncoding.EncodeToString([]byte(buffer.String()))

	// Create openstack vm
	if err := h.createServer(h.endpoint, h.token.ID, name, imageID, flavorID, h.networkID, ip, udata64, personnality); err != nil {
		return err
	}
	return nil
}

// Find image ID from image name
func (h *HatcheryCloud) imageID(img string) (string, error) {
	images, erri := getImages(h.endpoint, h.token.ID)
	if erri != nil {
		log.Warning("imageID> Error getting images: %s", erri)
	}

	for _, i := range images {
		if i.Name == img {
			return i.ID, nil
		}
	}
	return "", fmt.Errorf("imageID> image '%s' not found", img)
}

// Find flavor ID from flavor name
func (h *HatcheryCloud) flavorID(flavor string) (string, error) {
	for _, f := range h.flavors {
		if f.Name == flavor {
			return f.ID, nil
		}
	}
	return "", fmt.Errorf("flavor '%s' not found", flavor)
}

// Find network ID from network name
func (h *HatcheryCloud) getNetworkID(network string) (string, error) {
	for _, n := range h.networks {
		if n.Label == network {
			return n.ID, nil
		}
	}
	return "", fmt.Errorf("network '%s' not found", network)
}

// for each ip in the range, look for the first free ones
func (h *HatcheryCloud) findAvailableIP() (string, error) {
	var building, foundfree int

	servers := h.getServers()
	for _, s := range servers {
		if s.Status != "ACTIVE" {
			building++
		}
	}
	freeIP := []string{}
	for _, ip := range h.ips {
		free := true
		for _, s := range servers {
			if len(s.Addresses) == 0 {
				continue
			}
			for _, a := range s.Addresses[h.network] {
				if a.Addr == ip {
					free = false
				}
			}
			if !free {
				break
			}
		}
		if free {
			foundfree++
			if foundfree > building {
				freeIP = append(freeIP, ip)
			}
		}
	}

	if len(freeIP) == 0 {
		return "", fmt.Errorf("No IP left")
	}
	return freeIP[rand.Intn(len(freeIP))], nil
}

func (h *HatcheryCloud) updateServerList() {

	var out string
	var active, building, total int
	for _, s := range h.getServers() {
		out += fmt.Sprintf("- [%s] %s:%s (", s.Updated, s.Status, s.Name)
		for network, addr := range s.Addresses {
			out += fmt.Sprintf("%s:%s", network, addr[0].Addr)
		}
		out += fmt.Sprintf(")")
		switch s.Status {
		case serverStatusBuild:
			building++
		case serverStatusActive:
			active++
		}
		total++
	}
	log.Notice("Got %d servers (%d actives, %d booting)", total, active, building)
	log.Debug(out)

}

//////////// OPENSTACK HANDLERS //////////

type auth struct {
	Auth struct {
		Tenant string `json:"tenantName"`
		Creds  struct {
			User     string `json:"username"`
			Password string `json:"password"`
		} `json:"passwordCredentials"`
	} `json:"auth"`
}

// AccessType describe the access given by token
type AccessType struct {
	Token          Token                 `json:"token"`
	User           interface{}           `json:"id"`
	ServiceCatalog []ServiceCatalogEntry `json:"servicecatalog"`
}

// AuthToken is a specific openstack format
type AuthToken struct {
	Access AccessType `json:"access"`
}

// Token represent an openstack token
type Token struct {
	ID      string    `json:"id"`
	Expires time.Time `json:"expires"`
	Project struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"tenant"`
}

// ServiceCatalogEntry is an openstack specific object
type ServiceCatalogEntry struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Endpoints []ServiceEndpoint `json:"endpoints"`
}

// ServiceEndpoint describe an openstack endpoint
type ServiceEndpoint struct {
	Type        string `json:"type"`
	Region      string `json:"region"`
	PublicURL   string `json:"publicurl"`
	AdminURL    string `json:"adminurl"`
	InternalURL string `json:"internalurl"`
	VersionID   string `json:"versionid"`
}

// Link to downloadable resource
type Link struct {
	HRef string `json:"href"`
	Rel  string `json:"rel"`
}

// Network datastruct in openstack API
type Network struct {
	ID      string `json:"id,omitempty"`
	Label   string `json:"label,omitempty"`
	UUID    string `json:"uuid"`
	FixedIP string `json:"fixed_ip,omitempty"`
}

// Image datastruct in openstack API
type Image struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Links []Link `json:"links"`
}

// Flavor datastruct in openstack API
type Flavor struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Links []Link `json:"links"`
}

// Server datastruct in openstack API
type Server struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	ImageRef    string               `json:"imageRef"`  // The image reference, as a UUID or full URL
	FlavorRef   string               `json:"flavorRef"` // The flavor reference, as a UUID or full URL
	UserData    string               `json:"user_data"` // Scripts to use upon launch. Must be Base64 encoded.
	Metadata    map[string]string    `json:"metadata"`
	Networks    []Network            `json:"networks"`
	Links       []Link               `json:"links"`
	Status      string               `json:"status"`
	KeyName     string               `json:"key_name"`
	AccessIPv4  string               `json:"accessIPv4,omitempty"`
	Addresses   map[string][]Address `json:"addresses"`
	Updated     string               `json:"updated"`
	Personality Personality          `json:"personality"`
}

// Address datastruct in openstack API
type Address struct {
	Addr string `json:"addr"`
	Type string `json:"OS-EXT-IPS:type"`
}

// Personality is an array of files that are injected into the server at launch.
type Personality []*File

// File is used to inject a file into the server at launch.
// File implements the json.Marshaler interface, so when a Create is requested,
// json.Marshal will call File's MarshalJSON method.
type File struct {
	// Path of the file
	Path string
	// Contents of the file. Maximum content size is 255 bytes.
	Contents []byte
}

// MarshalJSON marshals the escaped file, base64 encoding the contents.
func (f *File) MarshalJSON() ([]byte, error) {
	file := struct {
		Path     string `json:"path"`
		Contents string `json:"contents"`
	}{
		Path:     f.Path,
		Contents: base64.StdEncoding.EncodeToString(f.Contents),
	}
	return json.Marshal(file)
}

func (h *HatcheryCloud) createServer(endpoint, token, name, image, flavor, network, ip, udata string, personality Personality) error {
	log.Notice("Create server %s %s", name, ip)
	uri := fmt.Sprintf("%s/servers", endpoint)

	s := Server{
		Name:        name,
		ImageRef:    image,
		FlavorRef:   flavor,
		UserData:    udata,
		Metadata:    map[string]string{"worker": name, "hatcheryName": h.Hatchery().Name},
		Personality: personality,
	}

	s.Networks = append(s.Networks, Network{UUID: network, FixedIP: ip})
	body := struct {
		Server Server `json:"server"`
	}{
		Server: s,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", uri, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	resp, err := hatchery.Client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		rbody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("cannot read body")
		}
		return unmarshalOpenstackError(rbody, resp.Status)
	}

	return nil
}

func deleteServer(endpoint, token, serverID string) error {
	uri := fmt.Sprintf("%s/servers/%s", endpoint, serverID)
	req, err := http.NewRequest("DELETE", uri, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	resp, err := hatchery.Client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		rbody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("cannot read body")
		}
		return unmarshalOpenstackError(rbody, resp.Status)
	}

	return nil
}

func getFlavors(endpoint string, token string) ([]Flavor, error) {
	uri := fmt.Sprintf("%s/flavors", endpoint)
	req, errRequest := http.NewRequest("GET", uri, nil)
	if errRequest != nil {
		return nil, errRequest
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	resp, errDo := hatchery.Client.Do(req)
	if errDo != nil {
		return nil, errDo
	}

	if resp.StatusCode >= 400 {
		rbody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("cannot read body")
		}
		return nil, unmarshalOpenstackError(rbody, resp.Status)
	}

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read body")
	}

	s := struct {
		Flavors []Flavor `json:"flavors"`
	}{}

	err = json.Unmarshal(rbody, &s)
	if err != nil {
		return nil, err
	}

	return s.Flavors, nil
}

//This a embeded cache for servers list
var servers = struct {
	mu   sync.RWMutex
	list []Server
}{
	mu:   sync.RWMutex{},
	list: []Server{},
}

func (h *HatcheryCloud) getServers() []Server {
	t := time.Now()

	defer log.Debug("getServers() : %d s", time.Since(t).Seconds())

	servers.mu.RLock()
	nbServers := len(servers.list)
	servers.mu.RUnlock()

	if nbServers == 0 {
		s, err := h.getServersRequest()
		if err != nil {
			log.Warning("getServers> error: %s", err)
		}
		servers.mu.Lock()
		servers.list = s
		servers.mu.Unlock()
		//Remove data from the cache after 2 seconds
		go func() {
			time.Sleep(2 * time.Second)
			servers.mu.Lock()
			servers.list = []Server{}
			servers.mu.Unlock()
		}()
	}

	return servers.list
}

func (h *HatcheryCloud) getServersRequest() ([]Server, error) {
	uri := fmt.Sprintf("%s/servers/detail", h.endpoint)
	req, errRequest := http.NewRequest("GET", uri, nil)
	if errRequest != nil {
		return nil, errRequest
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", h.token.ID)

	resp, errDo := hatchery.Client.Do(req)
	if errDo != nil {
		return nil, errDo
	}

	if resp.StatusCode >= 400 {
		rbody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("cannot read body")
		}
		return nil, unmarshalOpenstackError(rbody, resp.Status)
	}

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read body")
	}

	var s struct {
		Servers []Server `json:"servers"`
	}

	if err = json.Unmarshal(rbody, &s); err != nil {
		return nil, err
	}

	// Remove servers without "worker" tag
	var servers []Server
	for _, s := range s.Servers {
		_, worker := s.Metadata["worker"]
		if !worker {
			continue
		}
		workerHatcheryName, _ := s.Metadata["hatcheryName"]
		if workerHatcheryName == "" || workerHatcheryName != h.Hatchery().Name {
			continue
		}
		servers = append(servers, s)
	}
	return servers, nil
}

func getNetworks(endpoint string, token string) ([]Network, error) {
	uri := fmt.Sprintf("%s/os-networks", endpoint)
	req, errRequest := http.NewRequest("GET", uri, nil)
	if errRequest != nil {
		return nil, errRequest
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	resp, errDo := hatchery.Client.Do(req)
	if errDo != nil {
		return nil, errDo
	}

	if resp.StatusCode >= 400 {
		rbody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("cannot read body")
		}
		return nil, unmarshalOpenstackError(rbody, resp.Status)
	}

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read body")
	}

	s := struct {
		Networks []Network `json:"networks"`
	}{}

	if err = json.Unmarshal(rbody, &s); err != nil {
		return nil, err
	}

	return s.Networks, nil
}

func getImages(endpoint string, token string) ([]Image, error) {
	uri := fmt.Sprintf("%s/images", endpoint)
	req, errRequest := http.NewRequest("GET", uri, nil)
	if errRequest != nil {
		return nil, errRequest
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	resp, errDo := hatchery.Client.Do(req)
	if errDo != nil {
		return nil, errDo
	}

	if resp.StatusCode >= 400 {
		rbody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("cannot read body")
		}
		return nil, unmarshalOpenstackError(rbody, resp.Status)
	}

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read body")
	}

	s := struct {
		Images []Image `json:"images"`
	}{}

	if err = json.Unmarshal(rbody, &s); err != nil {
		return nil, err
	}

	return s.Images, nil
}

func getToken(user string, password string, url string, project string, region string) (*Token, string, error) {
	var endpoint string

	a := auth{}
	a.Auth.Tenant = project
	a.Auth.Creds.User = user
	a.Auth.Creds.Password = password

	data, err := json.Marshal(a)
	if err != nil {
		log.Critical("getToken> Marshal> %s", err)
		return nil, endpoint, err
	}

	uri := fmt.Sprintf("%s/v2.0/tokens", url)
	req, err := http.NewRequest("POST", uri, bytes.NewReader(data))
	if err != nil {
		log.Critical("getToken> newRequest> %s", err)
		return nil, endpoint, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(user, password)

	resp, err := hatchery.Client.Do(req)
	if err != nil {
		log.Critical("getToken> Do> %s", err)
		return nil, endpoint, err
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "json") != true {
		log.Critical("getToken> contains> %s", err)
		return nil, endpoint, fmt.Errorf("err (%s): header Content-Type is not JSON (%s)", contentType, resp.Status)
	}

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Critical("getToken> readall> %s", err)
		return nil, endpoint, fmt.Errorf("cannot read body")
	}

	if resp.StatusCode >= 400 {
		log.Critical("getToken> statuscode> user %s, password %s, url %s, project %s, region %s", user, password, url, project, region)
		return nil, endpoint, unmarshalOpenstackError(rbody, resp.Status)
	}

	var authRet AuthToken
	if err = json.Unmarshal(rbody, &authRet); err != nil {
		return nil, endpoint, err
	}

	for _, sc := range authRet.Access.ServiceCatalog {
		if sc.Name == "nova" {
			for _, e := range sc.Endpoints {
				if e.Region == region {
					endpoint = sc.Endpoints[0].PublicURL
				}
			}
		}
	}

	if endpoint == "" {
		return nil, "", fmt.Errorf("Nova endpoint in %s not found", region)
	}

	return &authRet.Access.Token, endpoint, nil
}

/*{"error": {"message": "The request you have made requires authentication.", "code": 401, "title": "Unauthorized"}}*/
type openstackError struct {
	Error struct {
		Message string `json:"error"`
		Code    int    `json:"code"`
		Title   string `json:"title"`
	} `json:"error"`
}

func unmarshalOpenstackError(data []byte, status string) error {
	operror := openstackError{}
	log.Warning("unmarshalOpenstackError> Error: %s", data)

	if err := json.Unmarshal(data, &operror); err != nil {
		return fmt.Errorf("%s", status)
	}

	if operror.Error.Code == 0 {
		return fmt.Errorf("%s", status)
	}

	return fmt.Errorf("%d: %s", operror.Error.Code, operror.Error.Message)
}

// IPinRanges returns a slice of all IP in all given IP ranges
// i.e 72.44.1.240/28,72.42.1.23/27
func IPinRanges(IPranges string) ([]string, error) {
	var ips []string

	ranges := strings.Split(IPranges, ",")
	for _, r := range ranges {
		i, err := IPinRange(r)
		if err != nil {
			return nil, err
		}
		ips = append(ips, i...)
	}

	return ips, nil
}

// IPinRange returns a slice of all IP in given IP range
// i.e 10.35.11.240/28
func IPinRange(IPrange string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(IPrange)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip2 := ip.Mask(ipnet.Mask); ipnet.Contains(ip2); inc(ip2) {
		log.Notice("Adding %s to IP pool", ip2)
		ips = append(ips, ip2.String())
	}

	return ips, nil
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
