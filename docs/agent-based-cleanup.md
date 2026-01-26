 #Proposed Solution for "Agent-based Cleanup":

  To fulfill the user's request ("once this agent is done working then the sandbox expires and gets cleaned up"):

   1. Track Created Resources: We need to modify FluidAgent in fluid/internal/tui/agent.go to maintain a list of
      createdSandboxIDs []string.
   2. Implement Cleanup: Add a Close() or Cleanup() method to FluidAgent that iterates through this list and calls
      vmService.DestroySandbox().
   3. Trigger Cleanup: In fluid/cmd/fluid/main.go, update the tuiCmd.RunE or runTUI function to defer the agent's cleanup
      method.

  This ensures that when the user exits the interactive session (or when the agent finishes its "work" if we view the session
  as the unit of work), the resources are reaped.

  I will write the response based on this "Session-Based Cleanup" approach, as it directly addresses the user's scenario of
  using the CLI tool.

  I will also mention the "Janitor" approach again as a complementary safety net, but focus on the "Session Cleanup" as the
  primary answer to "once this agent is done".

  I will structure the response to:
   1. Confirm Understanding: "You want the cleanup to happen automatically when the agent finishes its session."
   2. The Solution (Session Cleanup): Explain how to modify FluidAgent to track and destroy its own messes.
   3. Code Example: Show the modifications for agent.go and main.go.

  I am ready to write the response.

  One small refinement: The user said "the sandbox expires". My "Session Cleanup" is immediate destruction. "Expiration"
  usually implies time. But "expires and gets cleaned up by the AI agent" suggests an event-driven cleanup. I will clarify that
  this is "Session Cleanup".

  I'll proceed.

  ---

  Response

  That makes perfect sense for a CLI tool. You want a "leave no trace" policy where the agent cleans up after itself when the
  session ends.

  Since fluid is an interactive CLI tool, the best place to handle this is by implementing a Session Cleanup mechanism. The
  agent should track every sandbox it creates and destroy them when the CLI exits.

  Step 1: Track Created Sandboxes

  Modify your FluidAgent struct in fluid/internal/tui/agent.go to keep a ledger of what it built.

```go
   1 type FluidAgent struct {
   2     // ... existing fields
   3
   4     // Track resources created during this session
   5     createdSandboxes []string
   6 }
   ```

  Then, update the createSandbox method to append to this list:
```go
   1 func (a *FluidAgent) createSandbox(ctx context.Context, args []string) (map[string]interface{}, error) {
   2     // ... existing logic to create sb ...
   3
   4     // Track it!
   5     a.createdSandboxes = append(a.createdSandboxes, sb.ID)
   6
   7     return result, nil
   8 }
```
  Step 2: Implement the Cleanup Method

  Add a Close or Cleanup method to FluidAgent that iterates through that list.
```go
    1 // Cleanup destroys all sandboxes created during this session.
    2 func (a *FluidAgent) Cleanup(ctx context.Context) error {
    3     if len(a.createdSandboxes) == 0 {
    4         return nil
    5     }
    6
    7     a.logger.Info("cleaning up session resources", "count", len(a.createdSandboxes))
    8
    9     for _, id := range a.createdSandboxes {
   10         // Check if it still exists before destroying
   11         if _, err := a.vmService.GetSandbox(ctx, id); err == nil {
   12             a.logger.Info("destroying session sandbox", "id", id)
   13             if _, err := a.vmService.DestroySandbox(ctx, id); err != nil {
   14                 a.logger.Error("failed to destroy sandbox", "id", id, "error", err)
   15                 // Continue trying to destroy others even if one fails
   16             }
   17         }
   18     }
   19
   20     // Clear the list
   21     a.createdSandboxes = nil
   22     return nil
   23 }
```
  Step 3: Trigger on Exit

  Finally, hook this into your application lifecycle in fluid/cmd/fluid/main.go. You want to ensure this runs even if the user
  hits Ctrl+C.

  Update the runTUI function:
```go
    1 func runTUI() error {
    2     // ... setup config and services ...
    3
    4     agent := tui.NewFluidAgent(cfg, dataStore, vmService, libvirtMgr)
    5
    6     // Ensure cleanup runs when the function exits (user quits TUI)
    7     defer func() {
    8         fmt.Println("Cleaning up session resources...")
    9         ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   10         defer cancel()
   11         if err := agent.Cleanup(ctx); err != nil {
   12             fmt.Printf("Warning: cleanup failed: %v\n", err)
   13         }
   14     }()
   15
   16     model := tui.NewModel("fluid", "local", "vm-agent", agent, cfg, configPath)
   17     return tui.Run(model)
   18 }
```
  Summary
  With this change:
   1. Usage: You run fluid tui, tell the agent "Create a sandbox and test this script."
   2. Action: The agent creates the VM, does the work, and reports back.
   3. Exit: When you quit the tool, it automatically detects the leftover VM and destroys it, ensuring your host machine stays
      clean.

   
   
   
   ---
  
    Yes, you are absolutely right to be worried. I have analyzed the codebase (fluid/cmd/fluid/main.go and
    fluid/internal/vm/service.go) and while the intent to expire VMs is stored in the database via the TTLSeconds field, there is
    no active process currently watching that field.
  
    If a user creates a sandbox with a 1-hour TTL, the record will just sit in the database and the VM will remain running in
    Libvirt indefinitely after that hour passes.
  
    The Missing Piece: The "Janitor"
  
    To fix this, we need to introduce a background worker (a "Janitor") that runs alongside the main application.
  
    Proposed Architecture
  
    You should create a new Janitor service that runs a simple loop:
  
     1. Check: Every minute, query the database for "expired" sandboxes.
         * Query: WHERE state IN ('RUNNING', 'STARTING') AND (created_at + ttl_seconds * interval '1 second') < NOW()
     2. Act: For each match, call the existing vmService.DestroySandbox(id) method.
  
    Implementation Draft
  
    Here is how you can implement this in a new file fluid/internal/janitor/janitor.go:
  ```go
      1 package janitor
      2
      3 import (
      4     "context"
      5     "fluid/internal/store"
      6     "fluid/internal/vm"
      7     "log/slog"
      8     "time"
      9 )
     10
     11 type Janitor struct {
     12     store     store.Store
     13     vmService *vm.Service
     14     logger    *slog.Logger
     15 }
     16
     17 func New(st store.Store, svc *vm.Service) *Janitor {
     18     return &Janitor{
     19         store:     st,
     20         vmService: svc,
     21         logger:    slog.Default().With("component", "janitor"),
     22     }
     23 }
     24
     25 // Start runs the cleanup loop in a blocking manner.
     26 func (j *Janitor) Start(ctx context.Context, interval time.Duration) {
     27     ticker := time.NewTicker(interval)
     28     defer ticker.Stop()
     29
     30     for {
     31         select {
     32         case <-ctx.Done():
     33             return
     34         case <-ticker.C:
     35             j.cleanup(ctx)
     36         }
     37     }
     38 }
     39
     40 func (j *Janitor) cleanup(ctx context.Context) {
     41     // 1. Find expired sandboxes
     42     // This requires adding a 'ListExpiredSandboxes' method to your Store interface
     43     expired, err := j.store.ListExpiredSandboxes(ctx)
     44     if err != nil {
     45         j.logger.Error("failed to list expired sandboxes", "error", err)
     46         return
     47     }
     48
     49     for _, sb := range expired {
     50         j.logger.Info("destroying expired sandbox", "id", sb.ID, "ttl", sb.TTLSeconds)
     51         if _, err := j.vmService.DestroySandbox(ctx, sb.ID); err != nil {
     52             j.logger.Error("failed to destroy expired sandbox", "id", sb.ID, "error", err)
     53         }
     54     }
     55 }
  ```
    Wiring It Up
  
    Then, in your fluid/cmd/fluid/main.go (or wherever your long-running server process lives, if you have one), you would
    initialize and start this:
  ```go
     1 // In initServices or main()
     2 janitorSvc := janitor.New(dataStore, vmService)
     3
     4 // Run in a goroutine so it doesn't block the main process
     5 go janitorSvc.Start(ctx, 1 * time.Minute)
``` 
    Note: Since fluid is currently a CLI tool, this background worker assumes you are running a persistent server (like
    fluid-remote or a local daemon). If you only run the CLI for short commands, you might need a fluid cleanup command that can
    be run via system cron
