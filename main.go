package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const Version = "v0.0.1"

type Config struct {
	CFApiToken   string `toml:"cf_api_token"`
	CFZoneID     string `toml:"cf_zone_id"`
	CFRecordName string `toml:"cf_record_name"`
	CFIPType     string `toml:"cf_ip_type"`
	AddRecordIfMissing bool   `toml:"add_record_if_missing"`
	Interval     int    `toml:"interval"`
	RetryCount   int    `toml:"retry_count"`
	GetIPv4URL   string `toml:"get_ipv4_url"`
	GetIPv6URL   string `toml:"get_ipv6_url"`
	Notify       bool   `toml:"notify"`
	TgApiUrl     string `toml:"tg_api_url"` // 将 TG_PROXY_URL 改为 TG_API_URL
	TGToken      string `toml:"tg_token"`
	TGChatID     string `toml:"tg_chat_id"`
	Debug        bool   `toml:"debug"`
	LogPath      string `toml:"log_path"`
	LogRetention int    `toml:"log_retention"` // 日志保留天数
}

type CfDDNS struct {
	Config Config
}

func logMessage(message string) {
	currentTime := time.Now().Format("2024-11-25 15:04:05")
	fmt.Printf("[%s] %s\n", currentTime, message)
}

func loadConfig() Config {
	confPath := "conf.toml"

	// 检查配置文件是否存在
	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		log.Printf("Config file not found. Creating a default config file at %s", confPath)
		createDefaultConfig(confPath)
	}

	// 读取配置文件
	data, err := os.ReadFile(confPath)
	if err != nil {
		logMessage(fmt.Sprintf("Error reading config: %v", err))
		os.Exit(1)
	}

	var config Config
	// 解析配置文件
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
	if config.TgApiUrl == "" {
		config.TgApiUrl = "https://api.telegram.org"
	}

	// 如果未设置 GetIPv4URL，留空使用默认值https://4.ipw.cn
	if config.GetIPv4URL == "" {
		config.GetIPv4URL = "https://4.ipw.cn"
	}

	return config
}

func createDefaultConfig(configPath string) {
	defaultConfig := `
# Cloudflare API配置
cf_api_token = "your_CF_API_TOKEN_here"  # Cloudflare API Token
cf_zone_id = "Your_CF_ZONE_ID_HERE"    # Cloudflare Zone ID
cf_record_name = "YOUR_DOMAIN_HERE"  # 要更新的记录名称

# IP类型，用于指定获取IPv4还是IPv6
cf_ip_type = "46"  # 支持值：4（仅更新 IPv4），6（仅更新 IPv6），46（同时更新 IPv4 和 IPv6）

# 如果 DNS 记录不存在，是否自动添加
add_record_if_missing = true

# 执行间隔，单位为秒
interval = 60  # 每1分钟执行一次

# IP获取重试次数
retry_count = 3

# 获取IPv4地址的URL
get_ipv4_url = "https://4.ipw.cn"

# 获取IPv6地址的URL
get_ipv6_url = "https://6.ipw.cn"
# 获取公网 IPv4 和 IPv6 地址的 URL,备选
# get_ipv4_url = "https://api64.ipify.org"
# get_ipv6_url = "https://api6.ipify.org"

# Telegram配置
# 变动推送通知,1通知，0不通知
notify = false
tg_api_url = ""  # 自定义 Telegram API URL，如果不需要，留空
tg_token = "Your_tg_bot_token_here"
tg_chat_id = "Your_tg_chat_id_here"

# 调试模式
debug = false

# 日志设置
log_path = ''
# 日志保存时间，默认7天
log_retention = 7

`
	// 写入默认配置文件
	err := os.WriteFile(configPath, []byte(defaultConfig), 0644)
	if err != nil {
		log.Fatalf("Failed to create default config file: %v", err)
	}

	log.Printf("Default config file created at %s. Please review and update it as needed.", configPath)
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
			time.Sleep(2 * time.Second)
		} else if resp.StatusCode != 200 {
			lastError = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			logMessage(fmt.Sprintf("Attempt %d: Failed to retrieve IP address from %s. Status code: %d", i+1, url, resp.StatusCode))
			time.Sleep(2 * time.Second)
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
	if cf.Config.Notify {
		notifyMessage := fmt.Sprintf("Failed to retrieve IP address from %s after %d attempts. Last error: %v", url, retryCount, lastError)
		logMessage(notifyMessage)
		cf.tgMsg(notifyMessage)
	}

	os.Exit(1) // 可根据需求选择是否退出
	return ""
}

func (cf *CfDDNS) displayPublicIP() {
	// 获取 IPv4 地址
	ipv4, ipv4Err := cf.getPublicIP(cf.Config.GetIPv4URL)
	// 获取 IPv6 地址
	ipv6, ipv6Err := cf.getPublicIP(cf.Config.GetIPv6URL)

	// 输出结果
	if ipv4Err == nil {
		logMessage(fmt.Sprintf("IPv4 Address: %s", ipv4))
	} else {
		logMessage(fmt.Sprintf("Failed to get IPv4 Address: %v", ipv4Err))
	}

	if ipv6Err == nil {
		logMessage(fmt.Sprintf("IPv6 Address: %s", ipv6))
	} else {
		logMessage(fmt.Sprintf("Failed to get IPv6 Address: %v", ipv6Err))
	}

	// 判断优先级
	// if ipv4Err == nil && ipv6Err == nil {
	// 	logMessage("Current network priority: IPv6 > IPv4")
	// } else if ipv6Err == nil {
	// 	logMessage("Current network priority: IPv6")
	// } else if ipv4Err == nil {
	// 	logMessage("Current network priority: IPv4")
	// } else {
	// 	logMessage("No available network connection.")
	// }
}

// 获取公网 IP 地址的辅助函数
func (cf *CfDDNS) getPublicIP(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch IP from %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("non-200 status code from %s: %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	return strings.TrimSpace(string(body)), nil
}

func displayCloudflareIPPriority() {
	// 通过 Cloudflare 获取 IP 信息
	url := "https://cloudflare.com/cdn-cgi/trace"
	resp, err := http.Get(url)
	if err != nil {
		logMessage(fmt.Sprintf("Failed to fetch Cloudflare trace: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logMessage(fmt.Sprintf("Non-200 status code from Cloudflare trace: %d", resp.StatusCode))
		return
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logMessage(fmt.Sprintf("Failed to read Cloudflare trace response: %v", err))
		return
	}

	// 解析响应内容
	traceInfo := parseCloudflareTrace(string(body))
	if traceInfo == nil {
		logMessage("Failed to parse Cloudflare trace response.")
		return
	}

	// 输出 IP 信息和优先级
	ip := traceInfo["ip"]
	// logMessage(fmt.Sprintf("Detected Public IP: %s", ip))

	if isIPv6(ip) {
		logMessage("Current network priority: IPv6")
	} else if isIPv4(ip) {
		logMessage("Current network priority: IPv4")
	} else {
		logMessage("Unknown IP type. Unable to determine network priority.")
	}
}

// 辅助函数：解析 Cloudflare trace 响应
func parseCloudflareTrace(body string) map[string]string {
	traceInfo := make(map[string]string)
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			traceInfo[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return traceInfo
}

// 辅助函数：判断是否为 IPv6
func isIPv6(ip string) bool {
	return strings.Contains(ip, ":") && net.ParseIP(ip) != nil
}

// 辅助函数：判断是否为 IPv4
func isIPv4(ip string) bool {
	return strings.Contains(ip, ".") && net.ParseIP(ip) != nil
}



func (cf *CfDDNS) getCurrentDNSRecordIP(ipType string) map[string]string {
	result := make(map[string]string)

	// 如果 CF_IP_TYPE 是 46，同时获取 IPv4 和 IPv6
	ipTypes := []string{ipType}
	if ipType == "46" {
		ipTypes = []string{"4", "6"}
	}

	for _, t := range ipTypes {
		recordType := "A"
		if t == "6" {
			recordType = "AAAA"
		}

		// 构建请求头
		headers := map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", cf.Config.CFApiToken),
			"Content-Type":  "application/json",
		}

		// 获取当前 DNS 记录
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
			logMessage(fmt.Sprintf("Error fetching DNS record (%s): %v", t, err))
			result[t] = "Error fetching record"
			continue
		}
		defer resp.Body.Close()

		var apiResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
			logMessage(fmt.Sprintf("Error decoding DNS record response (%s): %v", t, err))
			result[t] = "Error decoding record"
			continue
		}

		// 获取 IP 地址
		if records, ok := apiResponse["result"].([]interface{}); ok && len(records) > 0 {
			record := records[0].(map[string]interface{})
			if content, ok := record["content"].(string); ok {
				result[t] = content
			} else {
				result[t] = "No content found"
			}
		} else {
			result[t] = "Record not found"
		}
	}

	return result
}

func (cf *CfDDNS) updateDNSRecord(ipType string) {
	ipTypes := []string{ipType}

	// 如果 CF_IP_TYPE 为 46，同时更新 IPv4 和 IPv6
	if ipType == "46" {
		ipTypes = []string{"4", "6"}
	}
	// 获取当前的 DNS 记录 IP（IPv4 和 IPv6）
	currentIPs := cf.getCurrentDNSRecordIP(ipType)
	for _, t := range ipTypes {
		ip := cf.getIP(t)
		currentIP, ok := currentIPs[t]
		if !ok {
			currentIP = "Unknown" // 如果返回的 map 中不存在对应类型，设置为未知
		}

		if ip == currentIP {
			logMessage(fmt.Sprintf("IPv%s: %s has not changed, no update needed.", t, currentIP))
			continue
		}
		updateResult := cf.updateDNSRecordHandle(t, cf.Config.CFRecordName, ip)

		// 发送 Telegram 通知
		if cf.Config.Notify {

			if updateResult {
				notificationMessage := fmt.Sprintf("IPv%s DNS record for %s updated from %s to %s successfully.", t, cf.Config.CFRecordName, currentIP, ip)
				cf.tgMsg(notificationMessage)
			} else {
				notificationMessage := fmt.Sprintf("IPv%s DNS record for %s updated from %s to %s failed.", t, cf.Config.CFRecordName, currentIP, ip)
				cf.tgMsg(notificationMessage)
			}
		}
	}

}

func (cf *CfDDNS) updateDNSRecordWithIP(ipType, ip string) {
	cf.updateDNSRecordHandle(ipType, cf.Config.CFRecordName, ip)
}

func (cf *CfDDNS) updateDNSRecordHandle(ipType, domainName string, ip string) bool {
	// 构建1个公共的DNS记录更新函数
	recordType := "A"
	if ipType == "6" {
		recordType = "AAAA"
	}

	// 构建请求头
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", cf.Config.CFApiToken),
		"Content-Type":  "application/json",
	}

	// 获取 DNS 记录 ID
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", cf.Config.CFZoneID)
	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	q := req.URL.Query()
	q.Add("name", domainName)
	q.Add("type", recordType)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logMessage(fmt.Sprintf("Error fetching DNS record: %v", err))
		return false
	}
	defer resp.Body.Close()

	var apiResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&apiResponse)

	var recordID string
	if records, ok := apiResponse["result"].([]interface{}); ok && len(records) > 0 {
		record := records[0].(map[string]interface{})
		recordID = record["id"].(string)
		// currentIP = record["content"].(string)
	} else if cf.Config.AddRecordIfMissing {
		// 如果记录不存在并且配置允许添加
		logMessage(fmt.Sprintf("DNS record IPv%s for %s not found. Adding a new record...", ipType, cf.Config.CFRecordName))
		recordID = cf.addDNSRecord(recordType, ip)
	}else {
		logMessage(fmt.Sprintf("DNS IPv%s record for %s not found.", ipType, domainName))
		return false
	}

	if recordID == "" {
		logMessage(fmt.Sprintf("Failed to find or create IPv%s DNS record for %s.", ipType, cf.Config.CFRecordName))
		return false
	}

	// 更新 DNS 记录
	data := map[string]interface{}{
		"type":    recordType,
		"name":    domainName,
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
		return false
	}
	logMessage(fmt.Sprintf("DNS IPv%s record for %s updated to %s successfully.", ipType, domainName, ip))
	return true
	// } else {
	// 	logMessage(fmt.Sprintf("DNS IPv%s record for %s not found.", ipType, domainName))
	// 	return false
	// }
}

// 添加 DNS 记录的辅助函数
func (cf *CfDDNS) addDNSRecord(recordType, ip string) string {
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", cf.Config.CFApiToken),
		"Content-Type":  "application/json",
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", cf.Config.CFZoneID)

	data := map[string]interface{}{
		"type":    recordType,
		"name":    cf.Config.CFRecordName,
		"content": ip,
		"ttl":     1800,
		"proxied": false,
	}

	body, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		logMessage(fmt.Sprintf("Failed to create DNS record (%s): %v", recordType, err))
		return ""
	}
	defer resp.Body.Close()

	var apiResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&apiResponse)

	if record, ok := apiResponse["result"].(map[string]interface{}); ok {
		if recordID, ok := record["id"].(string); ok {
			logMessage(fmt.Sprintf("Successfully created DNS record (%s) for %s.", recordType, cf.Config.CFRecordName))
			return recordID
		}
	}

	logMessage(fmt.Sprintf("Failed to parse response when creating DNS record (%s).", recordType))
	return ""
}

func (cf *CfDDNS) tgMsg(message string) {
	// 判断是否设置了自定义的 Telegram API URL
	// baseURL := "https://api.telegram.org"
	// if cf.Config.TgApiUrl != "" {
	// 	baseURL = cf.Config.TgApiUrl
	// }
	// baseURL := cf.Config.TgApiUrl

	// 构造完整的请求 URL
	url := fmt.Sprintf("%s/bot%s/sendMessage", cf.Config.TgApiUrl, cf.Config.TGToken)
	// logMessage(url)

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

// setupService 配置程序为系统服务
func setupService(serviceName string) {
	switch runtime.GOOS {
	// case "windows":
	// 	setupWindowsService(serviceName)
	case "linux":
		setupLinuxService(serviceName)
	default:
		logMessage("Service setup is not supported on this operating system.")
	}
}

// setupWindowsService 配置 Windows 服务
// func setupWindowsService(serviceName string) {
// 	m, err := mgr.Connect()
// 	if err != nil {
// 		logMessage(fmt.Sprintf("Failed to connect to Windows service manager: %v", err))
// 		return
// 	}
// 	defer m.Disconnect()

// 	exePath, err := os.Executable()
// 	if err != nil {
// 		logMessage(fmt.Sprintf("Failed to get executable path: %v", err))
// 		return
// 	}

// 	service, err := m.CreateService(serviceName, exePath, mgr.Config{
// 		StartType: mgr.StartAutomatic,
// 	})
// 	if err != nil {
// 		logMessage(fmt.Sprintf("Failed to create Windows service: %v", err))
// 		return
// 	}
// 	defer service.Close()

// 	logMessage(fmt.Sprintf("Windows service '%s' created successfully.", serviceName))
// }

// setupLinuxService 配置 Linux 服务
func setupLinuxService(serviceName string) {
	exePath, err := os.Executable()
	if err != nil {
		logMessage(fmt.Sprintf("Failed to get executable path: %v", err))
		return
	}

	serviceContent := `[Unit]
Description=CfDDNS Service
After=network.target

[Service]
ExecStart=%s
Restart=always
User=root

[Install]
WantedBy=multi-user.target
`
	serviceFile := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
	content := fmt.Sprintf(serviceContent, exePath)

	if err := os.WriteFile(serviceFile, []byte(content), 0644); err != nil {
		logMessage(fmt.Sprintf("Failed to write service file: %v", err))
		return
	}

	// 启用并启动服务
	cmds := [][]string{
		{"systemctl", "daemon-reload"},
		{"systemctl", "enable", serviceName},
		{"systemctl", "start", serviceName},
	}

	for _, cmd := range cmds {
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			logMessage(fmt.Sprintf("Failed to execute '%s %v': %v", cmd[0], cmd[1:], err))
			return
		}
	}

	logMessage(fmt.Sprintf("Linux service '%s' created and started successfully.", serviceName))
}

// removeService 移除系统服务
func removeService(serviceName string) {
	switch runtime.GOOS {
	// case "windows":
	// 	removeWindowsService(serviceName)
	case "linux":
		removeLinuxService(serviceName)
	default:
		logMessage("Service removal is not supported on this operating system.")
	}
}

// removeWindowsService 移除 Windows 服务
// func removeWindowsService(serviceName string) {
// 	m, err := mgr.Connect()
// 	if err != nil {
// 		logMessage(fmt.Sprintf("Failed to connect to Windows service manager: %v", err))
// 		return
// 	}
// 	defer m.Disconnect()

// 	service, err := m.OpenService(serviceName)
// 	if err != nil {
// 		logMessage(fmt.Sprintf("Service '%s' not found: %v", serviceName, err))
// 		return
// 	}
// 	defer service.Close()

// 	// 确认服务是否由本程序创建（简单示例，可扩展为更复杂校验）
// 	config, err := service.Config()
// 	if err != nil {
// 		logMessage(fmt.Sprintf("Failed to get service config: %v", err))
// 		return
// 	}

// 	if !strings.Contains(config.BinaryPathName, "cfddns") {
// 		logMessage(fmt.Sprintf("Service '%s' does not appear to be created by this program.", serviceName))
// 		logMessage(fmt.Sprintf("Service executable: %s", config.BinaryPathName))
// 		if !confirm("Do you want to remove this service anyway? (y/N)") {
// 			logMessage("Service removal canceled.")
// 			return
// 		}
// 	}

// 	// 删除服务
// 	err = service.Delete()
// 	if err != nil {
// 		logMessage(fmt.Sprintf("Failed to delete service '%s': %v", serviceName, err))
// 		return
// 	}

// 	logMessage(fmt.Sprintf("Service '%s' removed successfully.", serviceName))
// }

// removeLinuxService 移除 Linux 服务
func removeLinuxService(serviceName string) {
	serviceFile := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)

	// 检查服务文件是否存在
	if _, err := os.Stat(serviceFile); os.IsNotExist(err) {
		logMessage(fmt.Sprintf("Service file '%s' not found.", serviceFile))
		return
	}

	// 显示服务文件内容并确认
	content, err := os.ReadFile(serviceFile)
	if err != nil {
		logMessage(fmt.Sprintf("Failed to read service file '%s': %v", serviceFile, err))
		return
	}
	logMessage(fmt.Sprintf("Service file content:\n%s", string(content)))

	if !confirm("Do you want to remove this service? (y/N)") {
		logMessage("Service removal canceled.")
		return
	}

	// 停止并删除服务
	cmds := [][]string{
		{"systemctl", "stop", serviceName},
		{"systemctl", "disable", serviceName},
		{"rm", serviceFile},
		{"systemctl", "daemon-reload"},
	}

	for _, cmd := range cmds {
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			logMessage(fmt.Sprintf("Failed to execute '%s %v': %v", cmd[0], cmd[1:], err))
			return
		}
	}

	logMessage(fmt.Sprintf("Linux service '%s' removed successfully.", serviceName))
}

// confirm 显示确认提示
func confirm(message string) bool {
	fmt.Println(message)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func showHelp() {
	helpMessage := `
CfDDNS - Cloudflare Dynamic DNS Updater

This is a free software.
You can get it free here:
https://github.com/aircross/cfddns

                     |||                                                         
           ||||     |||||||||||||      |||||||||      |||      |||||     |||     
         |||||||    |    ||||||||||     ||||||||||      ||       |||   ||||||    
        |||    ||  ||     |||    |||     |||    |||     |||      ||   |||   ||   
       |||     ||  ||     |||     |||    |||     |||    ||||     ||   |||   ||   
       ||          ||||   |||      ||    |||      ||    |||||    ||    |||       
      |||         |||||   |||      |||   |||      |||   || |||   ||     ||||     
      |||          ||     |||       ||   |||       ||   ||   ||  ||       |||    
      |||          ||     |||       |    |||       |    ||    || ||         ||   
      |||          ||     |||      ||    |||      ||    ||     ||||   ||     |   
      |||          ||     |||      ||    |||      ||    ||      |||   ||    ||   
       ||       |  ||     |||     ||     |||     ||     |||      ||   ||||||||   
        ||     || ||||| |||||||||||    |||||||||||    ||||||      |    ||||||    
         ||||||||                                                                                                       

Usage:
  cfddns [command] [arguments]

Commands:
  tgtest              Send a test message to the configured Telegram chat.
  ip                  Query and display the current IP and network priority.
  now                 Query and display the current DNS record IP for the domain.
  v4 <IPv4>           Update the domain's IPv4 DNS record to the specified IPv4 address.
  v6 <IPv6>           Update the domain's IPv6 DNS record to the specified IPv6 address.
  v46                 Update the domain's IPv4 and IPv6 DNS record to the wan IP address.
  v, ver, version     Show the program version.
  h, help             Show this help message and exit.
Todo:
  s, service [name]   Set up the program as a system service. Default service name: cfddns.
  rs, removeservice [name]
                      Remove the specified system service. Default service name: cfddns.

Examples:
  cfddns              Run the program with the default configuration (dynamic DNS update).
  cfddns tgtest       Send a test message via Telegram.
  cfddns ip           Display the current IP address and network priority.
  cfddns now          Display the current IP address associated with the DNS record.
  cfddns v4           Update the domain's A record to wan IPv4 IP.
  cfddns v4 192.0.2.1 Update the domain's A record to 192.0.2.1.
  cfddns v6           Update the domain's A record to wan IPv6 IP.
  cfddns v6 2001:db8::1 Update the domain's AAAA record to 2001:db8::1.
  cfddns v46          Update the domain's A record to wan IPv4 and IPv6 IP.
  cfddns v            Show the program version.
  cfddns ver          Show the program version.
  cfddns version      Show the program version.
  cfddns help         Show this help message.
Todo: 
  cfddns s            Configure the program as a system service with the default name 'cfddns'.
  cfddns rs           Remove the program's system service with the default name 'cfddns'.
  cfddns rs myservice Remove the system service named 'myservice'.

Notes:
  - For commands like 'v4' and 'v6', the IP address must be valid, or an error will be shown.
  - Ensure the configuration file is properly set up before running the program.
  - system service Requires administrative privileges.
  - Services are registered differently on Windows and Linux.
  - Remove system service operation prompts if the service does not appear to be created by this program.
`
	fmt.Println(helpMessage)
}

func showVersion() {
	fmt.Printf("CfDDNS - Cloudflare Dynamic DNS Updater\nVersion: %s\n", Version)
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
		case "ip":
			cfddns.displayPublicIP()
			displayCloudflareIPPriority()
			// 
		case "now":
			// 查询并显示当前域名的 DNS 记录绑定的 IP
			logMessage("Fetching current DNS record IPs...")
			currentIPs := cfddns.getCurrentDNSRecordIP(cfddns.Config.CFIPType)
			for ipType, ip := range currentIPs {
				logMessage(fmt.Sprintf("Current DNS record IPv%s for %s: %s", ipType, cfddns.Config.CFRecordName, ip))
			}
		case "v4", "v6", "v46":
			if len(args) < 2 {
				ipType := args[0][1:] // 删除 "v" 前缀
				logMessage(fmt.Sprintf("Executing updateDNSRecord with IP type: %s", ipType))
				cfddns.updateDNSRecord(ipType)
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
		case "v", "ver", "version":
			// 显示版本信息
			showVersion()
		case "s", "service":
			// 设置服务
			serviceName := "cfddns"
			if len(args) > 1 {
				serviceName = args[1]
			}
			logMessage(fmt.Sprintf("Configuring service: %s", serviceName))
			setupService(serviceName)
		case "rs", "removeservice":
			// 移除服务
			serviceName := "cfddns"
			if len(args) > 1 {
				serviceName = args[1]
			}
			logMessage(fmt.Sprintf("Removing service: %s", serviceName))
			removeService(serviceName)
		default:
			logMessage(fmt.Sprintf("Unknown parameter: %s", args[0]))
			logMessage("Usage: cfddns [tgtest]")
		}
	} else {
		// 未传递参数，执行原逻辑
		cfddns.run()
	}
}
