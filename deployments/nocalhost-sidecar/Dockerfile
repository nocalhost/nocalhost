FROM nocalhost-docker.pkg.coding.net/nocalhost/public/minideb:master

MAINTAINER "Nocalhost"

# Install packages
RUN echo 'Acquire::Check-Valid-Until no;' > /etc/apt/apt.conf.d/99no-check-valid-until
RUN install_packages openssh-server wget
#COPY mutagen_linux_amd64_v0.11.8.tar.gz .
#RUN install_packages wget ca-certificates

RUN mkdir /root/.ssh && chmod 600 /root/.ssh && echo "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDqJOIfjQvv2pAanw3PBjpIqda+F7QAY0C4818D76C4u5Ybrja+Fz0cOCjtrRuwopsNcZhbGrva/zuG8J7Violft294fYVils7gOi1FjzA2twU1n90nCFpHt5uxETR9jR7JpsTUq15Xi6aIB5PynF/irr3EueUiiywhvzejbr1sA0ri26wteaSr/nLdNFy2TXVAEyHyzoxCAX4cECuGfarIgoQpdErc6dwyCh+lPnByL+AGP+PKsQmHmA/3NUUJGsurEf4vGaCd0d7/FGtvMG+N28C33Rv1nZi4RzWbG/TGlFleuvO8QV1zqIGQbUkqoeoLbbYsOW2GG0BxhJ7jqj9V root@eafa293b8956" > /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys
RUN mkdir -p /var/run/sshd && sed -ri 's/UsePAM yes/#UsePAM yes/g' /etc/ssh/sshd_config \
  && echo "UsePAM no\nAllowAgentForwarding yes\nPermitRootLogin yes\nPubkeyAuthentication yes\nAuthorizedKeysFile /root/.ssh/authorized_keys\n" >> /etc/ssh/sshd_config && echo "root:root" | chpasswd \
  && touch /root/.hushlogin \
  && true
RUN wget https://nocalhost-generic.pkg.coding.net/nocalhost/nhctl/mutagen_linux_amd64_v0.11.8.tar.gz --no-check-certificate && tar -xvzf mutagen_linux_amd64_v0.11.8.tar.gz && mv mutagen /usr/local/bin/ && rm mutagen_linux_amd64_v0.11.8.tar.gz mutagen-agents.tar.gz
#RUN tar -xvzf mutagen_linux_amd64_v0.11.8.tar.gz && mv mutagen /usr/local/bin/
#RUN rm mutagen_linux_amd64_v0.11.8.tar.gz mutagen-agents.tar.gz

EXPOSE 22

CMD ["sh","-c","service ssh start;/usr/local/bin/mutagen daemon start;tail -f /dev/null;"]
