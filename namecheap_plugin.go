package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// === Namecheap DNS Plugin ===

type NamecheapPlugin struct {
	name     string
	version  string
	apiURL   string
	apiKey   string
	username string
	clientIP string
}

func NewNamecheapPlugin() *NamecheapPlugin {
	return &NamecheapPlugin{
		name:    "namecheap",
		version: "1.0.0",
		apiURL:  "https://api.namecheap.com/xml.response",
	}
}

func (nc *NamecheapPlugin) Name() string {
	return nc.name
}

func (nc *NamecheapPlugin) Version() string {
	return nc.version
}

func (nc *NamecheapPlugin) Initialize(config map[string]interface{}) error {
	if apiKey, ok := config["api_key"].(string); ok {
		credentialMgr.StoreCredential("namecheap_api_key", apiKey)
		nc.apiKey = apiKey
	}

	if username, ok := config["username"].(string); ok {
		credentialMgr.StoreCredential("namecheap_username", username)
		nc.username = username
	}

	if clientIP, ok := config["client_ip"].(string); ok {
		saveConfig("namecheap_client_ip", clientIP)
		nc.clientIP = clientIP
	}

	return nil
}

func (nc *NamecheapPlugin) Validate() error {
	if nc.apiKey == "" && nc.username == "" {
		return fmt.Errorf("namecheap credentials not configured")
	}
	return nil
}

func (nc *NamecheapPlugin) Execute(ctx context.Context, action string, payload interface{}) (interface{}, error) {
	// Load credentials
	if nc.apiKey == "" {
		key, _ := credentialMgr.GetCredential("namecheap_api_key")
		nc.apiKey = key
	}
	if nc.username == "" {
		user, _ := credentialMgr.GetCredential("namecheap_username")
		nc.username = user
	}

	switch action {
	case "list_domains":
		return nc.listDomains(ctx)
	case "get_dns_records":
		return nc.getDNSRecords(ctx, payload)
	case "set_dns_record":
		return nc.setDNSRecord(ctx, payload)
	case "delete_dns_record":
		return nc.deleteDNSRecord(ctx, payload)
	case "add_subdomain":
		return nc.addSubdomain(ctx, payload)
	case "get_subdomains":
		return nc.getSubdomains(ctx, payload)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (nc *NamecheapPlugin) Shutdown() error {
	return nil
}

// API Response Types

type NamecheapResponse struct {
	XMLName xml.Name `xml:"ApiResponse"`
	Status  string   `xml:"Status,attr"`
	Errors  struct {
		Error string `xml:"Error"`
	} `xml:"Errors"`
	CommandResponse interface{}
}

type DomainInfo struct {
	Name string `xml:"Name,attr"`
	ID   string `xml:"ID,attr"`
}

type DNSRecord struct {
	RecordID   string `xml:"RecordId,attr"`
	HostName   string `xml:"HostName,attr"`
	RecordType string `xml:"Type,attr"`
	Address    string `xml:"Address,attr"`
	TTL        string `xml:"TTL,attr"`
	MXPriority string `xml:"MXPriority,attr"`
}

// Actions

func (nc *NamecheapPlugin) listDomains(ctx context.Context) (interface{}, error) {
	params := url.Values{}
	params.Set("ApiUser", nc.username)
	params.Set("ApiKey", nc.apiKey)
	params.Set("UserName", nc.username)
	params.Set("Command", "namecheap.domains.getList")
	params.Set("ClientIp", nc.clientIP)

	resp, err := http.Get(nc.apiURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("api call failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Parse response (simplified)
	var domains []map[string]string
	// In production, use proper XML parsing
	// var apiResp NamecheapResponse
	// xml.Unmarshal(body, &apiResp)

	return map[string]interface{}{
		"domains": domains,
		"raw":     string(body),
	}, nil
}

type GetDNSRequest struct {
	Domain string `json:"domain"`
}

func (nc *NamecheapPlugin) getDNSRecords(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	domain := req["domain"].(string)

	params := url.Values{}
	params.Set("ApiUser", nc.username)
	params.Set("ApiKey", nc.apiKey)
	params.Set("UserName", nc.username)
	params.Set("Command", "namecheap.dns.getHosts")
	params.Set("Domain", domain)
	params.Set("ClientIp", nc.clientIP)

	resp, err := http.Get(nc.apiURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("api call failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	return map[string]interface{}{
		"domain":  domain,
		"records": string(body),
	}, nil
}

type SetDNSRequest struct {
	Domain     string `json:"domain"`
	HostName   string `json:"hostname"`
	RecordType string `json:"type"`
	Address    string `json:"address"`
	TTL        string `json:"ttl"`
}

func (nc *NamecheapPlugin) setDNSRecord(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	domain := req["domain"].(string)
	hostname := req["hostname"].(string)
	recordType := req["type"].(string)
	address := req["address"].(string)
	ttl := req["ttl"].(string)

	if ttl == "" {
		ttl = "1800"
	}

	params := url.Values{}
	params.Set("ApiUser", nc.username)
	params.Set("ApiKey", nc.apiKey)
	params.Set("UserName", nc.username)
	params.Set("Command", "namecheap.dns.setHosts")
	params.Set("Domain", domain)
	params.Set("HostName1", hostname)
	params.Set("RecordType1", recordType)
	params.Set("Address1", address)
	params.Set("TTL1", ttl)
	params.Set("ClientIp", nc.clientIP)

	resp, err := http.Get(nc.apiURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("api call failed: %v", err)
	}
	defer resp.Body.Close()

	// Store DNS record locally
	now := int64(0)
	db.Exec(`
		INSERT INTO dns_records (id, domain, hostname, record_type, address, ttl, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, fmt.Sprintf("dns_%d", now), domain, hostname, recordType, address, ttl, now)

	return map[string]string{"status": "created"}, nil
}

type DeleteDNSRequest struct {
	Domain   string `json:"domain"`
	RecordID string `json:"record_id"`
}

func (nc *NamecheapPlugin) deleteDNSRecord(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	recordID := req["record_id"].(string)

	// Delete from database
	db.Exec(`DELETE FROM dns_records WHERE id = ?`, recordID)

	return map[string]string{"status": "deleted"}, nil
}

type AddSubdomainRequest struct {
	Domain    string `json:"domain"`
	Subdomain string `json:"subdomain"`
	IPAddress string `json:"ip_address"`
}

func (nc *NamecheapPlugin) addSubdomain(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	domain := req["domain"].(string)
	subdomain := req["subdomain"].(string)
	ipAddress := req["ip_address"].(string)

	// Add A record for subdomain
	return nc.setDNSRecord(ctx, map[string]interface{}{
		"domain":   domain,
		"hostname": subdomain,
		"type":     "A",
		"address":  ipAddress,
		"ttl":      "1800",
	})
}

type GetSubdomainsRequest struct {
	Domain string `json:"domain"`
}

func (nc *NamecheapPlugin) getSubdomains(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	domain := req["domain"].(string)

	var subdomains []map[string]interface{}
	rows, _ := db.Query(`
		SELECT hostname, record_type, address, ttl FROM dns_records WHERE domain = ?
	`, domain)
	defer rows.Close()

	for rows.Next() {
		var hostname, recordType, address, ttl string
		rows.Scan(&hostname, &recordType, &address, &ttl)
		subdomains = append(subdomains, map[string]interface{}{
			"hostname":    hostname,
			"record_type": recordType,
			"address":     address,
			"ttl":         ttl,
		})
	}

	return map[string]interface{}{
		"domain":     domain,
		"subdomains": subdomains,
	}, nil
}
