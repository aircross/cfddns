package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	CFAPIToken   string `toml:"CF_API_TOKEN"`
	CFZoneID     string `toml:"CF_ZONE_ID"`
	CFRecordName string `toml:"CF_RECORD_NAME"`
	CFIPType     string `toml:"CF_IP_TYPE"`
	Interval     int    `toml:"INTERVAL"`
	GetIPv4URL   string `toml:"GET_IPv4_URL"`
	GetIPv6URL   string `toml:"GET_IPv6_URL"`
	TGToken      string `toml:"TG_TOKEN"`
	TGChatID     string `toml:"TG_CHAT_ID"`
	Notify       int    `toml:"NOTIFY"`
	RetryCount   int    `toml:"RETRY_COUNT"`
	TGAPIURL     string `toml:"TG_API_URL"` // 将 TG_PROXY_URL 改为 TG_API_URL
}

type CfDDNS struct {
	Config Config
}

func logMessage(message string) {
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] %s\n", currentTime, message)
}

func loadConfig() Config {
	confPath := "conf.toml"
	data, err := os.ReadFile(confPath)
	if err != nil {
		logMessage(fmt.Sprintf("Error reading config: %v", err))
		os.Exit(1)
	}

	var config Config
	err = toml.Unmarshal(data, &config)
	if err != nil {
		logMessage(fmt.Sprintf("Error parsing config: %v", err))
		os.Exit(1)
	}

	// 如果未配置 RetryCount，设置默认值
	if config.RetryCount == 0 {
		config.RetryCount = 3
	}

	// 如果未设置 TG_API_URL，留空使用默认值
	if config.TGAPIURL == "" {
		config.TGAPIURL = ""
	}

	return config
}

// 校验 IPv4 地址是否合法
func isValidIPv4(ip string) bool {
	return net.ParseIP(ip) != nil && regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`).MatchString(ip)
}

// 校验 IPv6 地址是否合法
func isValidIPv6(ip string) bool {
	return net.ParseIP(ip) != nil && strings.Contains(ip, ":")
}

func (cf *CfDDNS) getIP1(ipType string) string {
	url := cf.Config.GetIPv4URL
	if ipType == "6" {
		url = cf.Config.GetIPv6URL
	}

	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		logMessage(fmt.Sprintf("Failed to retrieve IP address from %s, error: %v", url, err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	ip, _ := io.ReadAll(resp.Body)
	return string(ip)
}

func (cf *CfDDNS) getIP(ipType string) string {
	url := cf.Config.GetIPv4URL
	if ipType == "6" {
		url = cf.Config.GetIPv6URL
	}

	retryCount := cf.Config.RetryCount // 获取配置中的重试次数
	var lastError error

	for i := 0; i < retryCount; i++ {
		resp, err := http.Get(url)
		if err != nil {
			lastError = err
			logMessage(fmt.Sprintf("Attempt %d: Failed to retrieve IP address from %s. Error: %v", i+1, url, err))
		} else if resp.StatusCode != 200 {
			lastError = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			logMessage(fmt.Sprintf("Attempt %d: Failed to retrieve IP address from %s. Status code: %d", i+1, url, resp.StatusCode))
		} else {
			defer resp.Body.Close()
			ip, _ := io.ReadAll(resp.Body)
			return string(ip)
		}

		// 如果是非最后一次重试，暂停一段时间
		if i < retryCount-1 {
			time.Sleep(2 * time.Second)
		}
	}

	// 如果所有重试都失败，发送 Telegram 通知并终止
	// 发送 Telegram 通知
	if cf.Config.Notify == 1 {
		notifyMessage := fmt.Sprintf("Failed to retrieve IP address from %s after %d attempts. Last error: %v", url, retryCount, lastError)
		logMessage(notifyMessage)
		cf.tgMsg(notifyMessage)
	}

	os.Exit(1) // 可根据需求选择是否退出
	return ""
}

func (cf *CfDDNS) getCurrentDNSRecordIP(ipType string) string {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", cf.Config.CFZoneID)
	recordType := "A"
	if ipType == "6" {
		recordType = "AAAA"
	}

	req, _ := http.NewRequest("GET", url, nil)
	// 构建请求头
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", cf.Config.CFAPIToken),
		"Content-Type":  "application/json",
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	// req.Header.Set("Authorization", "Bearer "+cf.Config.CFAPIToken)
	// req.Header.Set("Content-Type", "application/json")
	q := req.URL.Query()
	q.Add("name", cf.Config.CFRecordName)
	q.Add("type", recordType)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logMessage(fmt.Sprintf("Error fetching DNS record: %v", err))
		return ""
		// os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	// json.NewDecoder(resp.Body).Decode(&result)

	// if records, ok := result["result"].([]interface{}); ok && len(records) > 0 {
	// 	record := records[0].(map[string]interface{})
	// 	return record["content"].(string)
	// }
	// return ""
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logMessage(fmt.Sprintf("Error decoding DNS record response: %v", err))
		return ""
	}

	if records, ok := result["result"].([]interface{}); ok && len(records) > 0 {
		record := records[0].(map[string]interface{})
		return record["content"].(string)
	}

	return ""
}

func (cf *CfDDNS) updateDNSRecord(ipType string) {
	ipTypes := []string{ipType}

	// 如果 CF_IP_TYPE 为 10，同时更新 IPv4 和 IPv6
	if ipType == "10" {
		ipTypes = []string{"4", "6"}
	}
	for _, t := range ipTypes {
		ip := cf.getIP(t)
		currentIP := cf.getCurrentDNSRecordIP(t)

		if ip == currentIP {
			logMessage(fmt.Sprintf("IPv%s: %s has not changed, no update needed.", t, currentIP))
			continue
		}
		recordType := "A"
		if t == "6" {
			recordType = "AAAA"
		}

		// 获取 DNS 记录 ID
		url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", cf.Config.CFZoneID)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+cf.Config.CFAPIToken)
		req.Header.Set("Content-Type", "application/json")
		q := req.URL.Query()
		q.Add("name", cf.Config.CFRecordName)
		q.Add("type", recordType)
		req.URL.RawQuery = q.Encode()
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logMessage(fmt.Sprintf("Error fetching IPv%s DNS record: %v", t, err))
			continue
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if records, ok := result["result"].([]interface{}); ok && len(records) > 0 {
			record := records[0].(map[string]interface{})
			recordID := record["id"].(string)

			// 更新 DNS 记录
			data := map[string]interface{}{
				"type":    recordType,
				"name":    cf.Config.CFRecordName,
				"content": ip,
				"ttl":     1800,
				"proxied": false,
			}

			body, _ := json.Marshal(data)
			updateURL := fmt.Sprintf("%s/%s", url, recordID)
			updateReq, _ := http.NewRequest("PUT", updateURL, bytes.NewBuffer(body))
			updateReq.Header.Set("Authorization", "Bearer "+cf.Config.CFAPIToken)
			updateReq.Header.Set("Content-Type", "application/json")

			updateResp, err := http.DefaultClient.Do(updateReq)
			if err != nil || updateResp.StatusCode != 200 {
				logMessage(fmt.Sprintf("Failed to update IPv%s DNS record: %v", t, err))
				continue
			}
			logMessage(fmt.Sprintf("IPv%s DNS record for %s updated from %s to %s successfully.", t, cf.Config.CFRecordName, currentIP, ip))
			// 发送 Telegram 通知
			if cf.Config.Notify == 1 {
				notificationMessage := fmt.Sprintf("IPv%s DNS record for %s updated from %s to %s successfully.", t, cf.Config.CFRecordName, currentIP, ip)
				cf.tgMsg(notificationMessage)
			}
		} else {
			logMessage(fmt.Sprintf("IPv%s DNS record for %s not found.", t, cf.Config.CFRecordName))
		}
	}

}

func (cf *CfDDNS) updateDNSRecordWithIP(ipType, ip string) {
	recordType := "A"
	if ipType == "6" {
		recordType = "AAAA"
	}

	// 构建请求头
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", cf.Config.CFAPIToken),
		"Content-Type":  "application/json",
	}

	// 获取 DNS 记录 ID
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", cf.Config.CFZoneID)
	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	q := req.URL.Query()
	q.Add("name", cf.Config.CFRecordName)
	q.Add("type", recordType)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logMessage(fmt.Sprintf("Error fetching DNS record: %v", err))
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if records, ok := result["result"].([]interface{}); ok && len(records) > 0 {
		record := records[0].(map[string]interface{})
		recordID := record["id"].(string)

		// 更新 DNS 记录
		data := map[string]interface{}{
			"type":    recordType,
			"name":    cf.Config.CFRecordName,
			"content": ip,
			"ttl":     1800,
			"proxied": false,
		}

		body, _ := json.Marshal(data)
		updateURL := fmt.Sprintf("%s/%s", url, recordID)
		updateReq, _ := http.NewRequest("PUT", updateURL, bytes.NewBuffer(body))
		for k, v := range headers {
			updateReq.Header.Set(k, v)
		}

		updateResp, err := http.DefaultClient.Do(updateReq)
		if err != nil || updateResp.StatusCode != 200 {
			logMessage(fmt.Sprintf("Failed to update DNS record: %v", err))
			return
		}
		logMessage(fmt.Sprintf("DNS record for %s updated to %s successfully.", cf.Config.CFRecordName, ip))
	} else {
		logMessage(fmt.Sprintf("DNS record for %s not found.", cf.Config.CFRecordName))
	}
}

func (cf *CfDDNS) tgMsg(message string) {
	// 判断是否设置了自定义的 Telegram API URL
	baseURL := "https://api.telegram.org"
	if cf.Config.TGAPIURL != "" {
		baseURL = cf.Config.TGAPIURL
	}

	// 构造完整的请求 URL
	url := fmt.Sprintf("%s/bot%s/sendMessage", baseURL, cf.Config.TGToken)
	logMessage(url)

	// url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cf.Config.TGToken)

	data := map[string]interface{}{
		"chat_id":                  cf.Config.TGChatID,
		"text":                     message,
		"disable_web_page_preview": true,
	}

	body, _ := json.Marshal(data)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		logMessage(fmt.Sprintf("Failed to create Telegram request: %v", err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		logMessage(fmt.Sprintf("Failed to send Telegram message: %v, status code: %d", err, resp.StatusCode))
		return
	}
	defer resp.Body.Close()

	logMessage("Telegram notification sent successfully.")
}

func showHelp() {
	helpMessage := `
CfDDNS - Dynamic DNS Updater

Usage:
  cfddns [command] [arguments]

Commands:
  tgtest              Send a test message to the configured Telegram chat.
  now                 Query and display the current DNS record IP for the domain.
  v4 <IPv4>           Update the domain's IPv4 DNS record to the specified IPv4 address.
  v6 <IPv6>           Update the domain's IPv6 DNS record to the specified IPv6 address.
  h, help             Show this help message and exit.

Examples:
  cfddns              Run the program with the default configuration (dynamic DNS update).
  cfddns tgtest       Send a test message via Telegram.
  cfddns now          Display the current IP address associated with the DNS record.
  cfddns v4 192.0.2.1 Update the domain's A record to 192.0.2.1.
  cfddns v6 2001:db8::1 Update the domain's AAAA record to 2001:db8::1.
  cfddns help         Show this help message.

Notes:
  - For commands like 'v4' and 'v6', the IP address must be valid, or an error will be shown.
  - Ensure the configuration file is properly set up before running the program.
`
	fmt.Println(helpMessage)
}

func (cf *CfDDNS) run() {
	for {
		cf.updateDNSRecord(cf.Config.CFIPType)
		logMessage(fmt.Sprintf("Waiting %d seconds before the next check.", cf.Config.Interval))
		time.Sleep(time.Duration(cf.Config.Interval) * time.Second)
	}
}

func main() {
	config := loadConfig()
	cfddns := CfDDNS{Config: config}
	// 检查是否带参数运行
	args := os.Args[1:] // 获取命令行参数（排除程序本身的名称）

	if len(args) > 0 {
		// 如果传递了参数
		switch args[0] {
		case "tgtest":
			// 测试 Telegram 消息推送
			testMessage := "This is a test message from CfDDNS."
			logMessage("Executing Telegram test message...")
			cfddns.tgMsg(testMessage)
			logMessage("Test message sent successfully.")
		case "now":
			// 查询并显示当前域名的 DNS 记录绑定的 IP
			logMessage("Fetching current DNS record IP...")
			ip := cfddns.getCurrentDNSRecordIP(cfddns.Config.CFIPType)
			if ip != "" {
				logMessage(fmt.Sprintf("Current DNS record IP for %s: %s", cfddns.Config.CFRecordName, ip))
			} else {
				logMessage(fmt.Sprintf("Failed to fetch DNS record for %s.", cfddns.Config.CFRecordName))
			}
		case "v4", "v6":
			if len(args) < 2 {
				logMessage(fmt.Sprintf("Missing IP address argument for %s.", args[0]))
				os.Exit(1)
			}
			ip := args[1]
			if args[0] == "v4" && isValidIPv4(ip) {
				logMessage(fmt.Sprintf("Updating IPv4 record for %s to %s...", cfddns.Config.CFRecordName, ip))
				cfddns.updateDNSRecordWithIP("4", ip)
			} else if args[0] == "v6" && isValidIPv6(ip) {
				logMessage(fmt.Sprintf("Updating IPv6 record for %s to %s...", cfddns.Config.CFRecordName, ip))
				cfddns.updateDNSRecordWithIP("6", ip)
			} else {
				logMessage(fmt.Sprintf("Invalid IP address for %s: %s.", args[0], ip))
				os.Exit(1)
			}
		case "h", "help":
			// 显示帮助信息
			showHelp()
		default:
			logMessage(fmt.Sprintf("Unknown parameter: %s", args[0]))
			logMessage("Usage: cfddns [tgtest]")
		}
	} else {
		// 未传递参数，执行原逻辑
		cfddns.run()
	}
}
