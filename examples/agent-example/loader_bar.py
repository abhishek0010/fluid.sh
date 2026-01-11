import time
import sys
import threading

def progress_thread(stop_event, title="Working...", bar_len=30):
    start = time.time()
    while not stop_event.is_set():
        elapsed = time.time() - start
        spinner = "|/-\\"[int(elapsed * 5) % 4]
        timer = time.strftime("%H:%M:%S", time.gmtime(elapsed))
        sys.stdout.write(f"\r{title} {spinner}  {timer}")
        sys.stdout.flush()
        time.sleep(0.1)
    # final line on stop
    elapsed = time.time() - start
    sys.stdout.write(f"\r{title} done  {time.strftime('%H:%M:%S', time.gmtime(elapsed))}\n")
    sys.stdout.flush()

def run_blocking_with_loader(blocking_func, *args, **kwargs):
    stop_event = threading.Event()
    t = threading.Thread(target=progress_thread, args=(stop_event, kwargs.pop("title", "Working...")))
    t.start()
    try:
        print("\n")
        for k, val in kwargs.items():
            print(f"{k}: {val}")
        for val in args:
            print(val)
        print("\n")
        result = blocking_func(*args, **kwargs)  # runs synchronously here
        return result
    finally:
        stop_event.set()
        t.join()
