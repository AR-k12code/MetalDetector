// PWA bootstrap and kiosk/reporting logic migrated from legacy Chrome App
(function () {
	// Config - replace with your endpoint and options via constants only
	const SETTINGS = {
		// Server endpoint to receive report POSTs
		server: 'put your endpoint here',
		useAuth: false,
		intervalMinutes: 1,
		maxGeoAgeSec: 600
	};

	// Settings are fixed at build time via the SETTINGS constant above.

	// Utility
	const nowSec = () => Math.floor(Date.now() / 1000);

	// Simple UI error helper
	function showError(msg) {
		try {
			let host = document.querySelector('.info') || document.body;
			let el = document.getElementById('serial-errors');
			if (!el) {
				el = document.createElement('p');
				el.id = 'serial-errors';
				el.style.color = '#900';
				el.style.marginTop = '8px';
				host.appendChild(el);
			}
			// Append or replace message
			const existing = el.textContent ? el.textContent + ' ' : '';
			el.textContent = existing + String(msg);
		} catch (e) {
			console.warn('error display failed', e, msg);
		}
	}

	// Serial resolution: use managed Device Attributes API.
	let serialCache;
	async function resolveSerial() {
		// Must be secure context for managed APIs
		if (location.protocol !== 'https:' && location.hostname !== 'localhost') {
			showError('This app must be served over HTTPS for managed device APIs.');
		}

		// WICG Device Attributes API: navigator.managed.getSerialNumber()
		try {
			const managed = navigator && navigator.managed;
			if (managed && typeof managed.getSerialNumber === 'function') {
				try {
					const serial = await managed.getSerialNumber();
					if (serial) return serial;
					showError('Managed API returned empty serial.');
				} catch (e) {
					showError(`Managed API getSerialNumber failed: ${e && e.name ? e.name : 'Error'}${e && e.message ? ' - ' + e.message : ''}`);
				}
			} else {
				showError('navigator.managed API not available. Ensure the PWA is force-installed and Permissions-Policy allows device-attributes.');
			}
		} catch (e) {
			showError('Error probing navigator.managed: ' + (e && e.message ? e.message : e));
		}

		return 'UNKNOWN';
	}

	async function getSerial() {
		if (serialCache) return serialCache;
		serialCache = await resolveSerial();
		return serialCache;
	}

	// Render serial into page and to # part of URL
	async function setSerialText() {
		const el = document.getElementById('serial');
		if (!el) return;
		try {
			const s = await getSerial();
			if (s && s !== 'UNKNOWN') {
				// Update URL fragment
				location.hash = encodeURIComponent(s);
			}
			el.textContent = s && s !== 'UNKNOWN' ? s : 'Unavailable';
			if (!s || s === 'UNKNOWN') showError('Serial not available.');
		} catch (e) {
			el.textContent = 'Unavailable';
			showError('Error while setting serial: ' + (e && e.message ? e.message : e));
		}
	}

	// Geolocation helper (resolves null on failure)
	function getGeo(maxAgeSec) {
		return new Promise(resolve => {
			if (!('geolocation' in navigator)) return resolve(null);
			navigator.geolocation.getCurrentPosition(
				pos => resolve({
					geoTimestamp: Math.floor(pos.timestamp / 1000),
					latitude: pos.coords.latitude,
					longitude: pos.coords.longitude,
					accuracy: pos.coords.accuracy
				}),
				err => {
					console.log('geolocation failed', err);
					resolve(null);
				},
				{ enableHighAccuracy: true, timeout: 10_000, maximumAge: (maxAgeSec || 0) * 1000 }
			);
		});
	}

	// Report payload
	async function buildReport() {
		const geo = await getGeo(SETTINGS.maxGeoAgeSec);
		const sessionStart = Number(localStorage.getItem('sessionStart')) || nowSec();
		if (!localStorage.getItem('sessionStart')) localStorage.setItem('sessionStart', String(sessionStart));

		const payload = {
			timestamp: nowSec(),
			email: 'PWA',
			serial: await getSerial(),
			sessionStart
		};
		if (geo) Object.assign(payload, geo);
		return payload;
	}

	async function sendReport() {
		if (!SETTINGS.server) return; // silently skip if not configured
		try {
			const body = await buildReport();
			await fetch(SETTINGS.server, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(body)
			});
			console.log('report sent');
		} catch (e) {
			console.warn('report failed', e);
		}
	}

	// Alarming/reporting loop (web fallback using setInterval)
	function startReportingLoop() {
		sendReport();
		const minutes = Math.max(1, SETTINGS.intervalMinutes);
		setInterval(sendReport, minutes * 60 * 1000);
	}

	// Parallax init
	function initParallax() {
		try {
			new Parallax(document.getElementById('background'), { originX: 0, originY: 0 });
		} catch { }
	}

	// Service worker registration for PWA
	function registerSW() {
		if ('serviceWorker' in navigator) {
			navigator.serviceWorker.register('sw.js').catch(err => console.warn('SW register failed', err));
		}
	}

	// Kickoff
	window.addEventListener('DOMContentLoaded', () => {
		setSerialText();
		initParallax();
		registerSW();
		startReportingLoop();
	});
})();
