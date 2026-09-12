package podman

import (
	"fmt"
	"io"
	"strings"
)

// DNSMasqImage is the local tag of the dnsmasq image lerd-dns runs. The tag
// carries a version because the rebuild trigger everywhere is "does this tag
// exist": the older lerd-dnsmasq:local has no binary under the name Exec= asks
// for, so an install holding it has to build again rather than keep it.
const DNSMasqImage = "lerd-dnsmasq:v2"

// DNSMasqExec is the name the binary carries inside the image. Not "dnsmasq" on
// purpose: AppArmor attaches profiles by executable path and cannot tell the
// container's /usr/sbin/dnsmasq from the host's, and the host profile permits
// only `signal (receive) peer=libvirtd`, so podman's SIGTERM and SIGKILL are
// both denied and `podman rm -f` never stops the container.
const DNSMasqExec = "lerd-dnsmasq"

// DNSMasqBaseImage is the base the dnsmasq image is built from, and the only
// thing that build downloads.
const DNSMasqBaseImage = "docker.io/library/alpine:latest"

const dnsmasqContainerfile = "FROM " + DNSMasqBaseImage +
	"\nRUN apk add --no-cache dnsmasq && mv /usr/sbin/dnsmasq /usr/local/bin/" + DNSMasqExec + "\n"

// BuildDNSMasqImage builds the dnsmasq image, falling back when apk cannot
// resolve names. apk resolves from inside the build's own network namespace
// rather than through the host resolver that just pulled the base image, and
// on some rootless hosts nothing reaches a nameserver from in there at all,
// pinned servers included (#1537). So the last attempt drops the namespace and
// builds on the host network, which is the one that demonstrably resolved the
// base image a moment earlier. Joining the lerd network is not an alternative:
// a rootless build cannot attach to a named network.
func BuildDNSMasqImage(w io.Writer, nameservers []string) error {
	err := buildDNSMasq(w, nil, false)
	if err == nil {
		return nil
	}
	if len(nameservers) > 0 {
		fmt.Fprintf(w, "\nretrying with the host resolvers (%s)\n", strings.Join(nameservers, ", "))
		if err = buildDNSMasq(w, nameservers, false); err == nil {
			return nil
		}
	}
	fmt.Fprintf(w, "\nretrying on the host network\n")
	return buildDNSMasq(w, nil, true)
}

func buildDNSMasq(w io.Writer, nameservers []string, hostNetwork bool) error {
	args := []string{"build", "-t", DNSMasqImage}
	for _, ns := range nameservers {
		args = append(args, "--dns", ns)
	}
	if hostNetwork {
		args = append(args, "--network", "host")
	}
	cmd := Cmd(append(args, "-")...)
	cmd.Stdin = strings.NewReader(dnsmasqContainerfile)
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}
