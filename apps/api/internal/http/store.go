package httpx

import (
	"agent-control-plane/apps/api/internal/service"
)

var gatewaySvc *service.GatewayService
var sessionSvc *service.SessionService
var dashboardSvc *service.DashboardService
var approvalSvc *service.ApprovalService
var policySvc *service.PolicyService

type AppStore interface {
	service.GatewayStore
	service.SessionStore
	service.DashboardStore
	service.ApprovalStore
	service.PolicyStore
}

func SetStore(store AppStore) {
	gatewaySvc = service.NewGatewayService(store)
	sessionSvc = service.NewSessionService(store)
	dashboardSvc = service.NewDashboardService(store)
	approvalSvc = service.NewApprovalService(store)
	policySvc = service.NewPolicyService(store)
}
