From quilt/quilt
Maintainer Ethan J. Jackson

RUN apt-get update \
&& apt-get install -y --no-install-recommends \
   bash git python curl \
&& rm -rf /var/lib/apt/lists/*

Copy bin/* /bin/
Copy tests /tests
Copy config/id_rsa /root/.ssh/id_rsa
RUN chmod 0600 /root/.ssh/id_rsa

Entrypoint ["quilt-tester"]
