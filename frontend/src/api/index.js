import request from "@/utils/request";

export function login(username, password) {
  return request({
    url: "/auth/login",
    method: "post",
    data: { username, password },
  });
}

export function getCurrentUser() {
  return request({
    url: "/auth/me",
    method: "get",
  });
}

export function getTicketTypes() {
  return request({
    url: "/ticket-types",
    method: "get",
  });
}

export function getGates() {
  return request({
    url: "/gates",
    method: "get",
  });
}

export function sellTicket(data) {
  return request({
    url: "/tickets/sell",
    method: "post",
    data,
  });
}

export function refundTicket(ticketNo) {
  return request({
    url: "/tickets/refund",
    method: "post",
    data: { ticket_no: ticketNo },
  });
}

export function getTicket(ticketNo) {
  return request({
    url: `/tickets/${ticketNo}`,
    method: "get",
  });
}

export function searchTickets(params) {
  return request({
    url: "/tickets",
    method: "get",
    params,
  });
}

export function checkIn(data) {
  return request({
    url: "/check-in",
    method: "post",
    data,
  });
}

export function checkOut(data) {
  return request({
    url: "/check-out",
    method: "post",
    data,
  });
}

export function getInParkCount() {
  return request({
    url: "/in-park-count",
    method: "get",
  });
}

export function getCheckRecords(params) {
  return request({
    url: "/check-records",
    method: "get",
    params,
  });
}

export function getDashboard() {
  return request({
    url: "/dashboard",
    method: "get",
  });
}

export function getGateStats(params) {
  return request({
    url: "/stats/gates",
    method: "get",
    params,
  });
}

export function getHourlyStats(params) {
  return request({
    url: "/stats/hourly",
    method: "get",
    params,
  });
}

export function getDailyStats(params) {
  return request({
    url: "/stats/daily",
    method: "get",
    params,
  });
}

export function getTicketTypeStats(params) {
  return request({
    url: "/stats/ticket-types",
    method: "get",
    params,
  });
}

export function getSlotHeatmap(params) {
  return request({
    url: "/stats/slot-heatmap",
    method: "get",
    params,
  });
}

export function getUsers(params) {
  return request({
    url: "/users",
    method: "get",
    params,
  });
}
