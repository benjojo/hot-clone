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

## Notice

You should avoid using this tool if you can help it. LVM, MD Raid, and DRBD give you far better (and tested) methods to enable system snapshotting/real time backups.