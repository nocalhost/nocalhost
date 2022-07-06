#!/usr/bin/env sh

iptables -t nat -D PREROUTING -p tcp -j NOCALHOST_INBOUND 2>/dev/null

iptables -t nat -F NOCALHOST_INBOUND 2>/dev/null
iptables -t nat -X NOCALHOST_INBOUND 2>/dev/null

iptables -t nat -F NOCALHOST_REDIRECT 2>/dev/null
iptables -t nat -X NOCALHOST_REDIRECT 2>/dev/null

set -e
iptables -t nat -N NOCALHOST_REDIRECT
iptables -t nat -A NOCALHOST_REDIRECT -p tcp -m multiport --dport "${NOCALHOST_PORT}" -j REDIRECT --to-port 16006
iptables -t nat -N NOCALHOST_INBOUND
iptables -t nat -A NOCALHOST_INBOUND -p tcp -j NOCALHOST_REDIRECT
iptables -t nat -A PREROUTING -p tcp -j NOCALHOST_INBOUND
