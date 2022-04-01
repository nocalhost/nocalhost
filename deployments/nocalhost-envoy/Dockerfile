FROM envoyproxy/envoy:v1.21.0

COPY envoy.yaml /etc/envoy/envoy.yaml
COPY docker-entrypoint.sh /
RUN chmod +x /docker-entrypoint.sh
COPY iptables.sh /
RUN chmod +x /iptables.sh

RUN apt-get update -y \
    && apt-get install iptables net-tools curl vim -y \
    && apt-get remove --purge --auto-remove -y && rm -rf /var/lib/apt/lists/*