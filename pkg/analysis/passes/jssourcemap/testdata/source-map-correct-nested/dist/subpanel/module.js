define(["@grafana/data","react","@emotion/css","@grafana/ui"],((e,t,o,a)=>(()=>{"use strict";var r={644:e=>{e.exports=o},305:t=>{t.exports=e},388:e=>{e.exports=a},650:e=>{e.exports=t}},n={};function s(e){var t=n[e];if(void 0!==t)return t.exports;var o=n[e]={exports:{}};return r[e](o,o.exports,s),o.exports}s.n=e=>{var t=e&&e.__esModule?()=>e.default:()=>e;return s.d(t,{a:t}),t},s.d=(e,t)=>{for(var o in t)s.o(t,o)&&!s.o(e,o)&&Object.defineProperty(e,o,{enumerable:!0,get:t[o]})},s.o=(e,t)=>Object.prototype.hasOwnProperty.call(e,t),s.r=e=>{"undefined"!=typeof Symbol&&Symbol.toStringTag&&Object.defineProperty(e,Symbol.toStringTag,{value:"Module"}),Object.defineProperty(e,"__esModule",{value:!0})};var l={};return(()=>{s.r(l),s.d(l,{plugin:()=>i});var e=s(305),t=s(650),o=s.n(t),a=s(644),r=s(388);const n=()=>({wrapper:a.css`
      font-family: Open Sans;
      position: relative;
      edit: false;
    `,svg:a.css`
      position: absolute;
      top: 0;
      left: 0;
    `,textBox:a.css`
      position: absolute;
      bottom: 0;
      left: 0;
      padding: 10px;
    `}),i=new e.PanelPlugin((({options:e,data:t,width:s,height:l})=>{const i=(0,r.useTheme2)(),p=(0,r.useStyles2)(n);return o().createElement("div",{className:(0,a.cx)(p.wrapper,a.css`
          width: ${s}px;
          height: ${l}px;
        `)},o().createElement("svg",{className:p.svg,width:s,height:l,xmlns:"http://www.w3.org/2000/svg",xmlnsXlink:"http://www.w3.org/1999/xlink",viewBox:`-${s/2} -${l/2} ${s} ${l}`},o().createElement("g",null,o().createElement("circle",{style:{fill:i.colors.primary.main},r:100}))),o().createElement("div",{className:p.textBox},e.showSeriesCount&&o().createElement("div",null,"Number of series: ",t.series.length),o().createElement("div",null,"Text option value: ",e.text)))})).setPanelOptions((e=>e.addTextInput({path:"text",name:"Simple text option edit",description:"Description of panel option",defaultValue:"Default value of text input option"}).addBooleanSwitch({path:"showSeriesCount",name:"Show series counter",defaultValue:!1}).addRadio({path:"seriesCountSize",defaultValue:"sm",name:"Series counter size",settings:{options:[{value:"sm",label:"Small"},{value:"md",label:"Medium"},{value:"lg",label:"Large"}]},showIf:e=>e.showSeriesCount})))})(),l})()));
//# sourceMappingURL=module.js.map