from virsh_sandbox import VirshSandbox

API_BASE = "http://localhost:8080"
TMUX_BASE = "http://localhost:8081"

client = VirshSandbox(API_BASE, TMUX_BASE)

def main():
    session = client.sandbox.create_sandbox()
    print("Hello from test!")


if __name__ == "__main__":
    main()
