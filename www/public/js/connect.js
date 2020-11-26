const uiElem = document.getElementById('ui');
const urlElem = document.getElementById('url');
const iconElem = document.getElementById('icon');

function copyToClipboard() {
  const tmpInput = document.createElement('input');
  document.body.appendChild(tmpInput);
  tmpInput.value = urlElem.innerText;
  tmpInput.select();
  document.execCommand("copy");
  tmpInput.remove();

  iconElem.style.opacity = '1.0';
}

uiElem.onmouseleave = () => {
  iconElem.style.opacity = '0.2';
};
