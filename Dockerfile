# Start with busybox, but with libc.so.6
FROM busybox:ubuntu-14.04

# So that we can run as unprivileged user inside the container.
RUN echo 'nobody:x:99:99:nobody:/:/bin/sh' >> /etc/passwd

USER nobody

ADD greetbot /usr/bin/i3-greetbot

VOLUME ["/var/lib/i3-greetbot"]

ENTRYPOINT ["/usr/bin/i3-greetbot", "-channel=#i3", "-histogram_path=/var/lib/i3-greetbot/histogram.data"]
