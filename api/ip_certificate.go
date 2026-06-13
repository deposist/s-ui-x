package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/deposist/s-ui-x/service"

	"github.com/gin-gonic/gin"
)

// IpCertIssueRequest is the body of POST /ip-cert/issue.
type IpCertIssueRequest struct {
	IP          string `json:"ip"`
	Email       string `json:"email"`
	Port        int    `json:"port"`
	ApplyTarget string `json:"applyTarget"`
}

// IssueIpCert obtains a Let's Encrypt certificate for an IP address and applies
// it to the panel or an inbound TLS profile. Issuance is privileged.
func (a *ApiService) IssueIpCert(c *gin.Context) {
	if !a.requireTokenScopeAny(c, "ipcert", "admin", "write") {
		return
	}
	var req IpCertIssueRequest
	if err := json.Unmarshal([]byte(c.PostForm("data")), &req); err != nil {
		if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
			c.JSON(http.StatusBadRequest, Msg{Success: false, Msg: "ipcert: invalid request"})
			return
		}
	}
	req.IP = strings.TrimSpace(req.IP)
	if err := service.ValidateIssuableIP(req.IP); err != nil {
		jsonMsg(c, "ipcert", err)
		return
	}
	status, err := a.IpCertificateService.IssueNow(
		c.Request.Context(), req.IP, strings.TrimSpace(req.Email), req.Port, req.ApplyTarget, getHostname(c),
	)
	a.recordAudit(c, requestActor(c), "ip_cert_issued", "ipcert", service.AuditSeverityWarn, map[string]any{
		"ip":     req.IP,
		"target": req.ApplyTarget,
		"ok":     err == nil,
	})
	jsonObj(c, status, err)
}

// GetIpCertStatus returns the current managed IP certificate state.
func (a *ApiService) GetIpCertStatus(c *gin.Context) {
	if !a.requireTokenScopeAny(c, "ipcert", "admin", "read", "write") {
		return
	}
	status, err := a.IpCertificateService.GetStatus()
	jsonObj(c, status, err)
}
