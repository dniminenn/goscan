const GoScan = {
  FETCH_DATA_INTERVAL: 5000,

  init() {
    document.addEventListener('DOMContentLoaded', () => {
      this.loadingElement = document.querySelector('.loading');
      this.containerElement = document.querySelector('.container');
      this.footerElement = document.querySelector('.footer');
      this.headerElement = document.querySelector('.header');
      this.errorBannerElement = document.querySelector('.error-banner');

      this.lastUpdated = new Date();
      this.activeHosts = {};

      this.showLoading();
      this.fetchData();
      setInterval(() => this.updateLastUpdated(), 1000);
    });
  },

  showLoading() {
    this.loadingElement.style.opacity = 1;
  },

  hideLoading() {
    this.loadingElement.style.opacity = 0;
    this.containerElement.style.opacity = 1;
    this.footerElement.style.opacity = 1;
    this.headerElement.style.opacity = 1;
  },

  showError(message) {
    this.errorBannerElement.textContent = message;
    this.errorBannerElement.style.display = 'block';
  },

  hideError() {
    this.errorBannerElement.style.display = 'none';
  },

  updateLastUpdated() {
    if (this.loadingElement.style.opacity === '0') {
      const now = new Date();
      const secondsAgo = Math.floor((now - this.lastUpdated) / 1000);
      const totalActiveHosts = this.calculateTotalActiveHosts();
      if (secondsAgo < 1) {
        this.footerElement.innerHTML = `${totalActiveHosts} hosts active. updated just now.`;
      } else {
        this.footerElement.innerHTML = `${totalActiveHosts} hosts active. updated ${secondsAgo} second${secondsAgo > 1 ? 's' : ''} ago.`;
      }
    }
  },

  calculateTotalActiveHosts() {
    let total = 0;
    for (const networkInterface in this.activeHosts) {
      total += this.activeHosts[networkInterface].activeHosts.length;
    }
    return total;
  },

  updateDisplay() {
    for (const [networkInterface, networkData] of Object.entries(this.activeHosts)) {
      let networkEl = this.containerElement.querySelector(`[data-interface="${networkInterface}"]`);
      if (!networkEl) {
        networkEl = document.createElement('div');
        networkEl.className = 'network';
        networkEl.dataset.interface = networkInterface;
        this.containerElement.appendChild(networkEl);
      }
      networkEl.innerHTML = `
        <h2><i class="fas fa-network-wired"></i>: ${networkInterface}</h2>
        <p>MAC Address: <span class="value">${networkData.MACAddress}</span></p>
        <p>Total IPs Scanned: <span class="value">${networkData.TotalIPsScanned}</span></p>
        <p>Active Hosts: <span class="host-count">${networkData.activeHosts.length}</span></p>
        <div class="host-list">${networkData.activeHosts.join('<br/>')}</div>
      `;
    }
  },

  fetchData() {
    fetch('/all')
      .then(response => {
        if (!response.ok) {
          throw new Error('Network response was not ok');
        }
        return response.json();
      })
      .then(data => {
        this.hideError();
        for (const [networkInterface, networkData] of Object.entries(data)) {
          if (!this.activeHosts[networkInterface]) {
            this.activeHosts[networkInterface] = {
              MACAddress: networkData.MACAddress,
              TotalIPsScanned: networkData.TotalIPsScanned,
              activeHosts: []
            };
          }
          networkData.activeHosts.forEach(host => {
            if (!this.activeHosts[networkInterface].activeHosts.includes(host)) {
              this.activeHosts[networkInterface].activeHosts.push(host);
            }
          });
          this.activeHosts[networkInterface].activeHosts.sort((a, b) => {
            const aParts = a.split('.').map(Number);
            const bParts = b.split('.').map(Number);
            for (let i = 0; i < 4; i++) {
              if (aParts[i] < bParts[i]) return -1;
              if (aParts[i] > bParts[i]) return 1;
            }
            return 0;
          });
        }
        this.updateDisplay();
        this.lastUpdated = new Date();
        this.updateLastUpdated();
        this.hideLoading();
        setTimeout(() => this.fetchData(), this.FETCH_DATA_INTERVAL);
      })
      .catch(error => {
        console.error('Error fetching data:', error);
        this.showError('Error fetching data: ' + error.message);
        this.hideLoading();
        setTimeout(() => this.fetchData(), this.FETCH_DATA_INTERVAL);
      });
  }
};

GoScan.init();
