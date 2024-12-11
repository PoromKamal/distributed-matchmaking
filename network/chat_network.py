#!/usr/bin/env python

from mininet.net import Mininet
from mininet.node import Controller, RemoteController, OVSController
from mininet.node import CPULimitedHost, Host, Node
from mininet.node import OVSKernelSwitch, UserSwitch
from mininet.node import IVSSwitch
from mininet.cli import CLI
from mininet.log import setLogLevel, info
from mininet.link import TCLink, Intf
from subprocess import call
from mininet.util import waitListening

def connectToRootNS( network, switch, ip, routes ):
    """Connect hosts to root namespace via switch. Starts network.
      network: Mininet() network object
      switch: switch to connect to root namespace
      ip: IP address for root namespace node
      routes: host networks to route to"""
    # Create a node in root namespace and link to switch 0
    root = Node( 'root', inNamespace=False )
    intf = network.addLink( root, switch ).intf1
    root.setIP( ip, intf=intf )
    # Start network that now includes link to root namespace
    network.start()
    # Add routes from root ns to hosts
    for route in routes:
        root.cmd( 'route add -net ' + route + ' dev ' + str( intf ) )
        
def sshd( network, cmd='/usr/sbin/sshd', opts='-D',
          ip='10.123.123.1/32', routes=None, switch=None ):
    """Start a network, connect it to root ns, and run sshd on all hosts.
       ip: root-eth0 IP address in root namespace (10.123.123.1/32)
       routes: Mininet host networks to route to (10.0/24)
       switch: Mininet switch to connect to root namespace (s1)"""
    if not switch:
        switch = network[ 's1' ]  # switch to use
    if not routes:
        routes = [ '10.0.0.0/24' ]
    connectToRootNS( network, switch, ip, routes )
    for host in network.hosts:
        host.cmd( cmd + ' ' + opts + '&' )
    info( "*** Waiting for ssh daemons to start\n" )
    for server in network.hosts:
        waitListening( server=server, port=22, timeout=5 )

    info( "\n*** Hosts are running sshd at the following addresses:\n" )
    for host in network.hosts:
        info( host.name, host.IP(), '\n' )
    info( "\n*** Type 'exit' or control-D to shut down network\n" )
    CLI( network )
    for host in network.hosts:
        host.cmd( 'kill %' + cmd )
    network.stop()
    
    
def myNetwork():

    net = Mininet( topo=None,
                   build=False,
                   ipBase='10.0.0.0/8')

    info( '*** Adding controller\n' )
    info( '*** Add switches\n')
    s3 = net.addSwitch('s3', cls=OVSKernelSwitch, failMode='standalone')
    s1 = net.addSwitch('s1', cls=OVSKernelSwitch, failMode='standalone')
    s6 = net.addSwitch('s6', cls=OVSKernelSwitch, failMode='standalone')
    s4 = net.addSwitch('s4', cls=OVSKernelSwitch, failMode='standalone')
    s2 = net.addSwitch('s2', cls=OVSKernelSwitch, failMode='standalone')
    s5 = net.addSwitch('s5', cls=OVSKernelSwitch, failMode='standalone')

    info( '*** Add hosts\n')
    c2 = net.addHost('c2', cls=Host, ip='10.0.0.2', defaultRoute=None)
    serv2 = net.addHost('serv2', cls=Host, ip='10.0.0.3', defaultRoute=None)
    serv1 = net.addHost('serv1', cls=Host, ip='10.0.0.7', defaultRoute=None)
    c1 = net.addHost('c1', cls=Host, ip='10.0.0.1', defaultRoute=None)
    serv3 = net.addHost('serv3', cls=Host, ip='10.0.0.4', defaultRoute=None)
    serv4 = net.addHost('serv4', cls=Host, ip='10.0.0.5', defaultRoute=None)
    central = net.addHost('central', cls=Host, ip='10.0.0.6', defaultRoute=None)

    info( '*** Add links\n')
    net.addLink(c1, s6, cls=TCLink, bw=10, delay="10ms")
    net.addLink(s6, central, cls=TCLink,bw=10, delay="10ms")
    net.addLink(s6, serv1, cls=TCLink,bw=10, delay="10ms")
    net.addLink(s6, c2, cls=TCLink,bw=10, delay="10ms")
    net.addLink(s6, s5, cls=TCLink,bw=10, delay="10ms")
    net.addLink(s5, s4, cls=TCLink,bw=10, delay="10ms")
    net.addLink(s4, serv4, cls=TCLink,bw=10, delay="10ms")
    net.addLink(s6, s3, cls=TCLink,bw=10, delay="10ms")
    net.addLink(s3, serv3, cls=TCLink,bw=10, delay="10ms")
    net.addLink(s3, s2, cls=TCLink,bw=10, delay="10ms")
    net.addLink(s2, s1,cls=TCLink,bw=10, delay="10ms")
    net.addLink(s1, serv2, cls=TCLink, bw=10, delay="10ms")

    info( '*** Starting network\n')
    net.build()
    info( '*** Starting controllers\n')
    for controller in net.controllers:
        controller.start()

    info( '*** Starting switches\n')
    net.get('s3').start([])
    net.get('s1').start([])
    net.get('s6').start([])
    net.get('s4').start([])
    net.get('s2').start([])
    net.get('s5').start([])
    
    info (' *** Starting Central\n')
    central.cmd("cd /home/mininet/d58-final-project/Central && sudo ./build.sh --run &")
    servers = [serv1, serv2, serv3, serv4]
    for server in servers:
      server.cmd("cd /home/mininet/d58-final-project/Server && sudo ./build.sh --run &")
    info( '*** Start SSHd daemon on hosts\n')
    sshd( net )
    
if __name__ == '__main__':
    setLogLevel( 'info' )
    myNetwork()