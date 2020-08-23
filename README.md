# Raspberry

## Basic Installation

### Download

The first step is to download Raspbian from [raspberry.org](https://www.raspberrypi.org/downloads/raspbian/).

### Install

After this you MUST BE CAREFUL. Incorrect action can result in wiping your entire computer. These instructions deviate from the [official ones](https://www.raspberrypi.org/documentation/installation/installing-images/mac.md).

1. Connect the SD card to the laptop.
2. Using `Disk Utility.app` erase the SD and format its with `ExFat` so that file size isn't limited to 4GB and choose `GUID Partition Map` so ???. Name it whatever you want. Note, this step will delete all data on the SD card.
3. Run `diskutil list` to get a list of the drives. The one with the format, name, and size of the SD card (from step #2). It should be called `/dev/disk6`, though the number at the end will likely be different.
4. Unmount the SD card using `diskutil unmountDisk /dev/disk6`.
5. Install the Raspbian image to the SD card using the following command where `raspbian.img` should be the path to the image you downloaded at the beginning. This may take a few minutes.

```
sudo dd bs=1m if=raspbian.img of=/dev/rdisk6 conv=sync
```

6. Eject the SD cars
```
sudo diskutil eject /dev/rdisk6
```

### Configure

1. Plug the SSD card, mouse, keyboard, and monitor into the Raspberry Pi.
2. Plug a power source in.
3. The red light should turn on and the green light should start flashing erratically.
4. Use the mouse to click through the dialogs.
5. Restart the Raspberry Pi if prompted.

### Enable SSH

#### GUI Approach

1. Navigate to Menu Bar > Navigate to Preferences > Raspberry Pi Configuration > Interfaces.
2. Enable SSH.
3. Click OK.

#### COMMAND LINE APPROACH

1. Enter `sudo raspi-config` in a terminal window
2. Select `Interfacing Options`
3. Navigate to and select `SSH`
4. Choose `Yes`
5. Select `Ok`
6. Choose `Finish`

#### TO SSH

To ssh, just run

```bash
ssh pi@10.0.0.241
```

[directions here [here](https://www.raspberrypi.org/documentation/remote-access/ssh/)]

### Install exFat support

The Linux Kernel doesn't support the exFat file system by default, so you have to install support by running the following in the command line:

```bash
sudo apt-get update
sudo apt-get install exfat-fuse exfat-utils
sudo mkdir /media/exfat
```

## Server Setup

### Golang

To install Go, simply run

```bash
sudo apt-get update
sudo apt-get install golang
```

Continuing via `y` whenever prompted. When the command is done, restart the command line application.

### Repo

Git clone this repo and cd into it. For a programatic interface, run

```bash
go run *.go /path/to/fileserving/root
```

The server supports three command line arguments (documented at the top of `main.go`):

```bash
go run *.go /path/to/file/serving/root -hd -ui -sleep
```

### Cronjob

Set up a cronjob by running

```bash
crontab -e
```

and add

```
@reboot go run /path/to/gitrepo/main.go /path/to/file/serving/root -hd -ui -sleep
```

If you named the machine `pi`, then two paths are probably `home/pi/raspberry/main.go` and `/media/pi/Untitled/videos`, respectively.

## Troubleshooting

### Lights

The Raspberry Pi has two lights.

The Red (PWR) light has three states:

- Off - no power
- Flashing - less than 4.65V
- On - 4.65V or more

Generally, the Green (ACT) light flashes "during SD card activity". It also has special patterns upon booting to indicate problems

- 3 flashes - `start.elf` not found
- 4 flashes - `start.elf` cannot launch (it is corrupt, the card is not correctly inserted, or the card slot isn't working)
- 7 flashes - `kernel.img` not found
- 8 flashes - `SDRAM` not recognized (it's probably damaged or either `bootcode/bin` or `start.elf` can't be read)

See [here](https://www.makeuseof.com/tag/raspberry-pi-wont-boot-fix/) for the source.