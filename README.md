hot-clone
===

This tool allows you to image an actively changing block device. Including the one the rootfs is stored on.

## Backup a device

hot-clone uses blktrace, and needs root to enable it. Performance impact seems to be low, though if you are planning to image a highly loaded disk you will want to change the values of:

```
  -blktrace.bufcount int
        The amount of buffers for blktrace to keep spare (default 16)
  -blktrace.bufsize int
        The size of each buffer for blktrace (default 65536)
```

To something higher, Ideally not too high since it will add latency to disk change events.

```
$ sudo ./hot-clone -device /dev/sdb > sdb.hc
[sudo] password for ben: 
2021/09/19 22:36:27 Read 18.0 MiB -- 0 Dirty sectors (0 event drops)
...
2021/09/19 22:39:48 Read 3.7 GiB -- 360 Dirty sectors (0 event drops)
2021/09/19 22:39:49 Catching up 36/360 sectors
2021/09/19 22:39:49 Catching up 72/360 sectors
2021/09/19 22:39:49 Catching up 108/360 sectors
2021/09/19 22:39:49 Catching up 144/360 sectors
2021/09/19 22:39:49 Catching up 180/360 sectors
2021/09/19 22:39:49 Catching up 216/360 sectors
2021/09/19 22:39:49 Catching up 252/360 sectors
2021/09/19 22:39:49 Catching up 288/360 sectors
2021/09/19 22:39:49 Catching up 324/360 sectors
2021/09/19 22:39:49 Catching up 360/360 sectors
2021/09/19 22:39:49 Done
```

The output it piped to stdout, this is so you can output to a file or a pipe (or directly invoke it from ssh)

## Restore a device

You cannot directly restore the output of hot-clone, you will first need to reassemble it. To do this you select `-reassemble` for the hot-clone output, and `-reassemble-output` to the device or file you wish to restore to.

```
$ ./hot-clone -reassemble sdb.hc -reassemble-output sdb.img
2021/09/19 22:40:14 Restoring section (Sector: 0 (len 3959422976 bytes)
$ 
```

## Compile from source code

Golang 1.13 or later is needed to compile. Debian 11 has a reccent enough version, compiling on older Debian will require manually installing golang.

Compiling on Debian 11 and copying the executable to a box running an older OS seems to work (on 64bit PC hardware).

To compile an executable binary:

On the github page, click on code, click on "Download ZIP".


```
$ unzip -e hot-clone-main.zip
$ cd hot-clone-main
$ go build

```
See if your executable runs.

```
$ ./hot-clone --help
  -blktrace.bufcount int
        The amount of buffers for blktrace to keep spare (default 16)
  -blktrace.bufsize int
        The size of each buffer for blktrace (default 65536)
  -device string
        The device you wise to hot-clone
  -print-writes
        print all writes happening
  -reassemble string
        use this hot-clone backup file to restore into a file or block device
  -reassemble-output string
        The path of the file or block device that is going to be restored to
  -verbose
        be extra verbose on whats happening

```
## Distributions

Brief testing indicates that this works on Debian 8, 9 and 11 and Ubuntu 18.10. 

Debian 7 does not have the TRACE interface so hot-clone won't work.

## Notice

You should avoid using this tool if you can help it. LVM, MD Raid, and DRBD give you far better (and tested) methods to enable system snapshotting/real time backups.
