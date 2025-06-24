// js/vars.js
/*const DIMENSIONS = [
  "date","ssp","network","publisher","appDomain","placement","transparency","tag","seller","device","mediatype","size","country","transaction","deal","dsp","buyer","brand","revenuetype"
];
const METRICS = [
  "requests","impressions","clicks","curatormargin","mediaspendext","revenuegross","revenuenet"
];
const METRIC_CHART_TYPE = {
  requests: "bar",
  impressions: "line",
  clicks: "bar",
  curatormargin: "line",
  mediaspendext: "bar",
  revenuegross: "line",
  revenuenet: "bar"
};*/
const DUMMY_VALUES = {
  date: (()=>{ let arr = []; let d = new Date(); d.setDate(d.getDate()-6); for(let i=0;i<7;i++) { arr.push(d.toISOString().slice(0,10)); d.setDate(d.getDate()+1);} return arr; })(),
  ssp: ["pubmatic","smart","appnexus"], network: ["PRISMA","MSQ","OpenX"], publisher: ["gala.fr","lefigaro.fr","rtl.fr"],
  appDomain: ["gala.fr","lefigaro.fr","rtl.fr"], placement: ["pave","header","footer"], transparency: ["transparent","opaque"],
  tag: ["pbs","openrtb"], seller: ["pubmatic","msq"], device: ["Webmob","Desktop","Tablet"], mediatype: ["banner","video"],
  size: ["300x250","728x90","320x50"], country: ["FR","US","UK"], transaction: ["open auction","deal"], deal: ["unknown","PMX123"],
  dsp: ["tradedesk","mediamath"], buyer: ["carrefour","unilever"], brand: ["coca","pepsi"], revenuetype: ["CPM","CPC"]
};
let selectedDimensions = [];
let selectedMetrics = [];
let filters = {};
let currentFilterDim = null;
let tempSelectedValues = [];
let comparisonPeriod = "";
let allResults = [];
let timeGrouping = "day";
