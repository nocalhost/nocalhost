FROM codingcorp-docker.pkg.coding.net/nocalhost/public/minideb:master

MAINTAINER "Nocalhost"

# Install packages
RUN echo 'Acquire::Check-Valid-Until no;' > /etc/apt/apt.conf.d/99no-check-valid-until
RUN install_packages openssh-server
COPY mutagen_linux_amd64_v0.11.8.tar.gz .
#RUN install_packages wget ca-certificates
RUN mkdir /root/.ssh && chmod 600 /root/.ssh && echo "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDqJOIfjQvv2pAanw3PBjpIqda+F7QAY0C4818D76C4u5Ybrja+Fz0cOCjtrRuwopsNcZhbGrva/zuG8J7Violft294fYVils7gOi1FjzA2twU1n90nCFpHt5uxETR9jR7JpsTUq15Xi6aIB5PynF/irr3EueUiiywhvzejbr1sA0ri26wteaSr/nLdNFy2TXVAEyHyzoxCAX4cECuGfarIgoQpdErc6dwyCh+lPnByL+AGP+PKsQmHmA/3NUUJGsurEf4vGaCd0d7/FGtvMG+N28C33Rv1nZi4RzWbG/TGlFleuvO8QV1zqIGQbUkqoeoLbbYsOW2GG0BxhJ7jqj9V root@eafa293b8956" > /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys
RUN mkdir -p /var/run/sshd \
  && echo "UsePAM no\nAllowAgentForwarding yes\nPermitRootLogin yes\nPermitRootLogin yes\nPubkeyAuthentication yes\nRSAAuthentication yes\nAuthorizedKeysFile /root/.ssh/authorized_keys\n" >> /etc/ssh/sshd_config && echo "root:root" | chpasswd \
  && touch /root/.hushlogin \
  && true
#RUN wget https://codingcorp-generic.pkg.coding.net/nocalhost/nhctl/mutagen_linux_amd64_v0.11.8.tar.gz && tar -xvzf mutagen_linux_amd64_v0.11.8.tar.gz && cp mutagen /usr/local/bin/

EXPOSE 22

CMD ["sh","-c","service ssh start;/usr/local/bin/mutagen daemon start;tail -f /dev/null;"]
