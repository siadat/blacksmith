# How it works

Blacksmith is forked from [Pixiecore](https://github.com/danderson/pixiecore).

## Pixiecore

Booting a Linux system over the network is quite tedious. You have to
set up a TFTP server, configure your DHCP server to recognize PXE
clients, and send them the right set of magical options to get them to
boot, often fighting rubbish PXE ROM implementations.

Pixiecore implements four different, but related protocols in one
binary, which together can take a PXE ROM from nothing to booting
Linux. They are: ProxyDHCP, PXE, TFTP, and HTTP. Let's walk through
the boot process for a PXE ROM.

## DHCP/ProxyDHCP

The first thing a PXE ROM does is request a configuration through
DHCP, waiting for a DHCP reply that includes PXE vendor options. At
this step, we try to get the assined `Machine` for the given hardware
address (mac), or create a new one, and pass the ip and the pxe params
to the PXE Rom.

Pixiecore has implemented a ProxyDHCP, which can co-exist with the
current DHCP server in the network. As it was a case for us, we have
changed it to a normal DHCP server.

## PXE

(TODO: As we're not using ProxyDHCP, can we optimse this step in our
code? Although this step takes less than a second in our experiments.)

In theory, you'd expect the ProxyDHCP server to just provide a TFTP
server IP and a filename to the PXE firmware, and it would proceed to
download and boot that just like the BOOTP of old.

Sadly, the average quality of PXE ROM implementations is abysmal, and
many of them fail to chainload correctly if you try to do this from a
ProxyDHCP server.

So, instead, we make use of the spec's "PXE menu" functionality, which
lets you tell the PXE firmware to display a boot menu. Just like
everything else in PXE, this is quite brittle, so nobody actually uses
it to display menus - instead, they just push a more fully featured
bootloader over PXE, and let that bootloader do the fancy work.

However, PXE menus seem to work reliably when combined with
ProxyDHCP... And the PXE configuration can provide a timeout after
which the first menu entry is booted... And that timeout can be set to
zero.

So, we can just provide a single-entry menu, with a zero timeout, and
chainload that way! But wait, there's more terribleness. PXE menu
entries don't just list a TFTP server and file to load, because that
would be too simple. Instead, each menu entry maps to a "Boot Server
Type", and yet another DHCP option maps that boot server type to a set
of IP addresses.

Those IP addresses aren't TFTP servers, but PXE boot servers. PXE boot
servers listen on port 4011. They use the DHCP packet format, but only
as a way of conveying a DHCP option that says "please tell me how to
boot the following Boot Server Type". It's quite possibly the least
efficient protocol encoding ever devised.

At long last, when the PXE server receives that request, it can reply
with a BOOTP-ish packet that specified next-server and a filename. And
_those_ are, at long last, TFTP.

## TFTP

After navigating the eldritch horror of PXE, TFTP is a breath of fresh
air. It is indeed a trivial protocol for transferring files. I have
found some PXE ROMs that manage to add unnecessary complexity even to
that, but by and large, this step is straightforward.

However, TFTP is quite slow, because it doesn't support transfer
windows (well, it does, but it's an extension defined in an RFC
published in 2015, so guess how many PXE ROMs implement it...). As a
result, you must pay one round-trip per ~1500 bytes transferred, and
even on a gigabit network, that slows things down.

Given that some netboot images are quite large (CoreOS clocks in at
almost 200MB), what we really want is to switch to a more efficient
protocol. That's where PXELINUX comes in.

PXELINUX is a small bootloader that knows how to boot Linux kernels,
and it comes in a variant that can speak HTTP. PXELINUX is 90kB, which
even over TFTP is very fast to transfer.

Thus, Pixiecore uses TFTP only to transfer PXELINUX, and from there
steers it to HTTP for the rest of the loading process.

## HTTP

We've finally crawled our way up to the late nineties - we can speak
HTTP! Pixiecore's HTTP server is wonderfully familiar and normal. It
just serves up a support file that PXELINUX needs (`ldlinux.c32`), a
trivial PXELINUX configuration telling it to boot a Linux kernel, and
the user-provided kernel and initrd files.

PXELINUX grabs all of that, and finally, Linux boots.

## Recap

This is what the whole boot process looks like on the wire.

### Dramatis Personae

- **Machine**, which want to join to our cluster.
  - **PXE ROM**, a brittle firmware burned into the network card of
  the Machine.
- **Blacksmith**, the server of DHCP, PXE, TFTP and two HTTPs!
- **PXELINUX**, an open source bootloader of the [Syslinux project][syslinux].

[syslinux]: http://www.syslinux.org

### Timeline

- Machine, the PXE ROM, broadcasts `DHCPDISCOVER`.
- Blacksmith's DHCP server responds with a `DHCPOFFER` containing
  network configs.
- Machine, and so PXE ROM starts, broadcasts `DHCPREQUEST`.
- Blacksmith's DHCP server responds with a `DHCPACK` containing a PXE
  boot menu.
- PXE ROM processes the PXE boot menu, decides to boot menu entry 0
  (after 2 seconds).
- PXE ROM sends a `DHCPREQUEST` to Blacksmith's PXE server, asking for
  a boot file.
- Pixiecore's PXE server responds with a `DHCPACK` listing a TFTP
  server, a boot filename, and a PXELINUX vendor option to make it use
  HTTP.
- PXE ROM downloads PXELINUX from Blacksmith's TFTP server, and hands
  off to PXELINUX.
- PXELINUX fetches its configuration from Blacksmith's HTTP server.
  The *bootparams*, included in this configuration. is customizable
  through Blacksmith workspace.
- PXELINUX fetches a kernel and ramdisk from Blacksmith's HTTP server,
  and boots CoreOS.
- CoreOS on Machine asks for a *cloudinit* (or *ignition*) file from
  another HTTP server on  Blacksmith.
- The cloudinit is generated according to the workspace and the states
  stored in the etcd datasource, and is returned to the Machine.
