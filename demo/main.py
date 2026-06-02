import tkinter as tk
from tkinter import ttk


class DemoApp(tk.Tk):
    def __init__(self):
        super().__init__()
        self.title("Pipeline CLI Demo")
        self.geometry("420x320")
        self.resizable(False, False)
        self.count = 0

        self._build_ui()

    def _build_ui(self):
        padding = {"padx": 16, "pady": 8}

        header = ttk.Label(
            self,
            text="Pipeline CLI Demo",
            font=("Segoe UI", 16, "bold"),
        )
        header.pack(**padding)

        subtitle = ttk.Label(
            self,
            text="Simple Python UI — run with: python main.py",
            wraplength=360,
        )
        subtitle.pack(padx=16, pady=(0, 12))

        greet_frame = ttk.LabelFrame(self, text="Say hello")
        greet_frame.pack(fill="x", padx=16, pady=8)

        self.name_entry = ttk.Entry(greet_frame)
        self.name_entry.pack(fill="x", padx=12, pady=(10, 6))
        self.name_entry.bind("<Return>", lambda _e: self.say_hello())

        ttk.Button(greet_frame, text="Say hello", command=self.say_hello).pack(
            padx=12, pady=(0, 10)
        )

        self.greet_label = ttk.Label(greet_frame, text="", wraplength=340)
        self.greet_label.pack(padx=12, pady=(0, 10))

        counter_frame = ttk.LabelFrame(self, text="Counter")
        counter_frame.pack(fill="x", padx=16, pady=8)

        row = ttk.Frame(counter_frame)
        row.pack(pady=12)

        ttk.Button(row, text="−", width=4, command=self.decrease).pack(side="left", padx=8)
        self.count_label = ttk.Label(row, text="0", font=("Segoe UI", 20, "bold"))
        self.count_label.pack(side="left", padx=12)
        ttk.Button(row, text="+", width=4, command=self.increase).pack(side="left", padx=8)

        self.status = ttk.Label(self, text="Status: Ready", foreground="#0a7")
        self.status.pack(pady=12)

    def say_hello(self):
        name = self.name_entry.get().strip()
        if name:
            self.greet_label.config(text=f"Hello, {name}! Welcome to the Pipeline CLI demo.")
        else:
            self.greet_label.config(text="Please enter your name first.")

    def increase(self):
        self.count += 1
        self.count_label.config(text=str(self.count))

    def decrease(self):
        self.count -= 1
        self.count_label.config(text=str(self.count))


def main():
    app = DemoApp()
    app.mainloop()


if __name__ == "__main__":
    main()
    