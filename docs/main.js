const regCode = /(RJ|VJ|BJ)\d{6}/g;
const endPoint = "https://dlsite.ster.email/dlsite/db/"

function updateInfo(rjcode, apiKey) {
    const xhr_info = new XMLHttpRequest();
    const xhr_img = new XMLHttpRequest();
    var sbutton = document.getElementById('buttonSearch');

    var loading = document.getElementById('loadingIcon');
    var loaded = document.getElementById('buttonSearchGo');

    xhr_img.open('GET', endPoint + rjcode + '/img', true);
    xhr_img.setRequestHeader('x-api-key', apiKey)
    xhr_img.responseType = 'text'
    xhr_img.onload = function() {
        if(xhr_img.status === 200) {
            var img = document.getElementById('workImage');

            img.src = "data:image/jpeg;base64,"+xhr_img.response
            img.onclick = function () {
                location.href = xhr_info.response.req_url;
            }
        } else {
            console.log("image request failed: "+xhr_img.status)
        }
        loading.style.display = 'none';
        loaded.style.display = 'block';
    }

    xhr_info.open('GET', endPoint + rjcode, true);
    xhr_info.setRequestHeader('x-api-key', apiKey)
    xhr_info.responseType = 'json';
    xhr_info.onload = () => {
        document.getElementById('workContent').innerHTML =
            'Title: ' + xhr_info.response.title + '<br>' +
            'Brand: ' + xhr_info.response.maker + '<br>' +
            'Release: ' + xhr_info.response.releaseDate
        xhr_img.send();
    }
    xhr_info.send();
    loading.style.display = 'inline-block';
    loaded.style.display = 'none';
}

window.onload = function () {
    var sbar = document.getElementById('barSearch');
    var sbutton = document.getElementById('buttonSearch');
    var apiKey = "";

    const URLSearch = new URLSearchParams(location.search);
    if(URLSearch.has('key')) {
        apiKey = URLSearch.get('key');
    }

    sbutton.onclick = function () {
        res = regCode.exec(sbar.value)
        if (res) {
            sbar.value = res[0];
            updateInfo(res[0], apiKey)
        }
    }
    sbar.onkeydown = function (e) {
        if (e.code == 'Enter') {
            res = regCode.exec(sbar.value)
            if (res) {
                sbar.value = res[0];
                updateInfo(res[0], apiKey)
            }
        }
    }

    
}