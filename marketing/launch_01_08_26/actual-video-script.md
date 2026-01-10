

---
## Script

**[Camera: Facing me at the library or in my room]**
**[I'm talking with the jlab microphone maybe]**

Hey, my name is Collin and I'm excited to introduce Fluid. Fluid is an open-source library that enables AI to safely work on your infrastructure.

Let me show you an example of Fluid in action:

**[Zed editor]**

Here I have a server running a httpd server that is hosting an html site.

I can access it on my laptop in real time.

I've been having tons of traffic recently and want to load balance with another server.

Here I have my agent starting up and creating a sandbox of the vm it needs to work on.

The sandbox has been created and now the agent has the ability to run commands, edit files, change systemctl and do whatever else it wants, all while building an ansible playbook in the background.

Once it verifies that the server is working from the load balancer, I get the option to run this ansible playbook and deploy it on the main VM. 

Let me look over the commandssssss. and looks good to me! I can view what commands it ran and how it moved throughout the sandbox. Afterward I can deploy this out and check the live changes. Boom, easy as that.

Now you don't have agents touching prod while also getting the benefits of autonomous work, check it out below.
