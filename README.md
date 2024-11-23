# CfDDNS - Cloudflare Dynamic DNS Updater
##### 这是一个设计用于利用CF的API TOKEN自动更新IP到Cloudflare,实现域名动态更行绑定的的小工具
##### 初衷是因为自己需要将自己家里的IPv6更新到CF，以便于可以快速地通过IPv6访问自己家里的电脑
##### 从一开始的网上找到尝试修改别人的，再到尝试用python手搓，到最后的golang手搓
##### 最终借助ChetGPT，以及到处搜到编写了本小工具
  
🔑 CF_API_TOKEN是你的Cloudflare API token
  
`CF_API_TOKEN`应该是API **token** (_不是_ API key), 你可以在后面的链接处生成 [API Tokens页面](https://dash.cloudflare.com/profile/api-tokens). 通过 **Edit zone DNS** 模板来创建1个 token. 
  
#### 使用说明如下：
```shell
CfDDNS  -  Cloudflare  Dynamic  DNS  Updater
  
Usage:
 cfddns [command] [arguments]
  
Commands:
 tgtest Send  a  test  message  to  the  configured  Telegram  chat.
 now  Query  and  display  the  current  DNS  record  IP  for  the  domain.
 v4  <IPv4>  Update  the  domain's IPv4 DNS record to the specified IPv4 address.
  v6 <IPv6>           Update the domain's  IPv6  DNS  record  to  the  specified  IPv6  address.
 v,  ver,  version  Show  the  program  version.
 h,  help  Show  this  help  message  and  exit.
Todo:
 s,  service [name]   Set up the program as a system service. Default service name: cfddns.
 rs,  removeservice [name]
 Remove  the  specified  system  service.  Default  service  name:  cfddns.
  
Examples:
 cfddns Run  the  program  with  the  default  configuration (dynamic DNS  update).
 cfddns  tgtest  Send  a  test  message  via  Telegram.
 cfddns  now Display  the  current  IP  address  associated  with  the  DNS  record.
 cfddns  v4  192.0.2.1  Update  the  domain's A record to 192.0.2.1.
  cfddns v6 2001:db8::1 Update the domain's  AAAA  record  to  2001:db8::1.
 cfddns  version Show  the  program  version.
 cfddns  help  Show  this  help  message.
Todo:
 cfddns  s Configure  the  program  as  a  system  service  with  the  default  name  'cfddns'.
 cfddns  rs  Remove  the  program's system service with the default name 'cfddns'.
  cfddns rs myservice Remove the system service named 'myservice'.
  
Notes:
  - For commands like 'v4' and 'v6', the IP address must be valid, or an error will be shown.
  - Ensure the configuration file is properly set up before running the program.
  - system service Requires administrative privileges.
  - Services are registered differently on Windows and Linux.
  - Remove system service operation prompts if the service does not appear to be created by this program.
```