const statusDot = document.getElementById("status-dot");
const statusText = document.getElementById("status-text");
const nameInput = document.getElementById("name-input");
const greetBtn = document.getElementById("greet-btn");
const greetOutput = document.getElementById("greet-output");
const countEl = document.getElementById("count");
const decBtn = document.getElementById("dec-btn");
const incBtn = document.getElementById("inc-btn");

let count = 0;

statusDot.classList.add("ok");
statusText.textContent = "Ready";

greetBtn.addEventListener("click", () => {
  const name = nameInput.value.trim();
  greetOutput.textContent = name
    ? `Hello, ${name}! Welcome to the Pipeline CLI demo.`
    : "Please enter your name first.";
});

nameInput.addEventListener("keydown", (e) => {
  if (e.key === "Enter") greetBtn.click();
});

function updateCount() {
  countEl.textContent = String(count);
}

decBtn.addEventListener("click", () => {
  count -= 1;
  updateCount();
});

incBtn.addEventListener("click", () => {
  count += 1;
  updateCount();
});
