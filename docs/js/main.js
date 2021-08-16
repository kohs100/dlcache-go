const regCode = /(RJ|VJ|BJ)\d{6}/g;
const endPoint = "https://dlsite.ster.email/dlsite/db/"

const objs = {
    img: null,
    category: null,
    date: null,
    maker: null,
    pid: null,
    title: null,
    ind: null
}

function loadStart() {
    objs.ind.innerText = '...';
}

function loadEnd() {
    objs.ind.innerText = 'Go';
}

function updateInfo(ctg, rls, mkr, pid, title) {
    objs.category.innerText = ctg;
    objs.date.innerText = rls;
    objs.maker.innerText = mkr;
    objs.pid.innerText = pid;
    objs.title.innerText = title;
}

function getInfo(rjcode, apiKey) {
    const xhr_info = new XMLHttpRequest();
    const xhr_img = new XMLHttpRequest();

    xhr_img.open('GET', endPoint + rjcode + '/img');
    xhr_img.setRequestHeader('x-api-key', apiKey)
    xhr_img.responseType = 'text'
    xhr_img.onload = function () {
        if (xhr_img.status === 200) {
            objs.img.src = "data:image/jpeg;base64," + xhr_img.response
            objs.img.onclick = function () {
                location.href = xhr_info.response.req_url;
            }
        } else if (xhr_img.status === 404) {
            objs.img.src = "notfound.png";
        } else {
            objs.img.src = "failed.png";
            console.log('Image Request Response State: ' + xhr_img.status.toString());
        }
        loadEnd();
    }

    xhr_info.open('GET', endPoint + rjcode, true);
    xhr_info.setRequestHeader('x-api-key', apiKey)
    xhr_info.responseType = 'json';
    xhr_info.onload = function () {
        if (xhr_info.status === 200) {
            updateInfo( xhr_info.response.category,
                        xhr_info.response.releaseDate,
                        xhr_info.response.maker,
                        xhr_info.response.productId,
                        xhr_info.response.title);
            xhr_img.send();
        } else if (xhr_info.status === 404) {
            updateInfo(" ", " ", " ", " ", "Not Found");
            objs.img.src = "notfound.png";
            loadEnd();
        } else {
            updateInfo("", "", "", "", "Failed");
            objs.img.src = "failed.png";
            console.log('Metadata Request Response State: ' + xhr_img.status.toString());
            loadEnd();
        }
    }
    xhr_info.send();
    loadStart();
}

window.onload = function () {
    var sbar = document.getElementById('srchField');
    var sbutton = document.getElementById('srchIndicator');
    var apiKey = "";

    objs.img = document.getElementById('workImage');
    objs.category = document.getElementById('CTGValue');
    objs.date = document.getElementById('RLSValue');
    objs.maker = document.getElementById('MKRValue');
    objs.pid = document.getElementById('PIDValue');
    objs.title = document.getElementById('titleBox');
    objs.ind = document.getElementById('srchIndicator');

    const URLSearch = new URLSearchParams(location.search);
    if (URLSearch.has('key')) {
        apiKey = URLSearch.get('key');
    }

    function start() {
        res = regCode.exec(sbar.value.toUpperCase())
        if (res) {
            sbar.value = res[0];
            getInfo(res[0], apiKey)
        }
    }

    sbutton.onclick = start;

    sbar.onkeydown = function (e) {
        if (e.code == 'Enter') start();
    }
}