'use strict';

document.addEventListener('DOMContentLoaded', () => {
	const btn = document.getElementById('copy-bas');
	const code = document.getElementById('bas-code');
	if (!btn || !code) return;

	btn.addEventListener('click', async () => {
		try {
			await navigator.clipboard.writeText(code.textContent);
			const orig = btn.textContent;
			btn.textContent = 'Copied!';
			btn.classList.add('copied');
			setTimeout(() => {
				btn.textContent = orig;
				btn.classList.remove('copied');
			}, 1500);
		} catch (err) {
			btn.textContent = 'Copy failed';
			setTimeout(() => { btn.textContent = 'Copy'; }, 1500);
		}
	});
});
