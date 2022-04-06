# remote-exec
A remote executor framework using go-rpc and ssh

# What is this
This framework tries to use golang to remotely excute code and ge the results back

It uses
* RPC
* SSH
to accomplish that.

# How does it work

There are two modes in which it operates:

* Client
* Server

In the client mode, you clone the binary --> to the remote server and start the server on the remote machine
the remote machine and the client machine are connected over an SSH tunnel.

Now you can execute RPC calls between them to accomplish remote tasks.

# Why do you want to do this

Executing system commands remotely is easy but automating them is not easy, you have to go through executing shell commands over SSH
This lets you use go code with much better error handling.

For example if you want to test if the remote machine is redhat verion 7 you will have to create a remote SSH connection and go through this:

```

	output, _, err := sshClient.RunCommand("awk -F\\\" '/VERSION_ID/ {print $2}' /etc/os-release")
	if err != nil {
		return osfiles, err
	}
	osfiles.osVersion = strings.TrimSpace(string(output))

```
With this package you can write an RPC Server and RPC Client to accomplish the same

RPC Server

```
type OSChecker int

func (o *OSChecker) CheckVersion(arg *Args, ret *String) error {

  var si sysinfo.SysInfo

	si.GetSysInfo()
  
  ret = si.GetVersion()
  return nil
  }
  

```

RPC Client

```
// Synchronous call
	args := &Args{}
	var reply string
	
	err := client.Call("OSChecker.CheckVersion", args, &reply)
	
```
