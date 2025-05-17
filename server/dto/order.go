package dto

var STKPushRequest struct {
	IspID        string `json:"isp_id"`
	Phone        string `json:"phone"`
	Username     string `json:"username"`
	PlanID       int    `json:"plan_id"`
	DeviceID     string `json:"device_id"`
	Zone         string `json:"zone"`
	IsHomeUser   bool   `json:"is_home_user"`
	DeviceCount  int     `json:"devices"`
	Mac          string `json:"mac"`
	Ip           string `json:"ip"`
}