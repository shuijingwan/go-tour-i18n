(() => {
  const nav=document.querySelector('[data-project-nav]');
  if(!nav)throw new Error('shared project navigation missing');
  const desktop=nav.parentElement, details=nav.querySelector('details'), summary=details.querySelector('summary');
  const media=matchMedia('(max-width:57.699rem)');
  // Only the Site host has a navigation drawer. Tour retains its compact
  // native header and course navigation; the language details stays there.
  const drawer=document.querySelector('.NavigationDrawer.js-header'), toggle=document.querySelector('.js-headerMenuButton');
  const drawerOpen=()=>media.matches&&!!drawer&&drawer.classList.contains('is-active');
  const move=()=>{details.open=false;if(drawer)(media.matches?drawer.querySelector('.NavigationDrawer-nav'):desktop).append(nav)};
  move();media.addEventListener('change',move);
  if(drawer)new MutationObserver(()=>{const open=drawer.classList.contains('is-active');toggle.setAttribute('aria-expanded',String(open));if(!open)details.open=false}).observe(drawer,{attributes:true,attributeFilter:['class']});
  // Consume both phases of an owned Escape. The Site drawer also handles
  // bubbling keyup; letting that phase through would dismiss both layers.
  let escapeKeyUpPending=false;
  document.addEventListener('keydown',e=>{
    if(e.key==='Escape'){
      if(escapeKeyUpPending&&e.repeat){e.preventDefault();e.stopPropagation();return}
      if(details.open){details.open=false;summary.focus()}
      else if(drawerOpen()){toggle.click();toggle.focus()}
      else return;
      escapeKeyUpPending=true;
      e.preventDefault();e.stopPropagation();
    }
    if(e.key==='Tab'&&drawerOpen()){
      const controls=[...drawer.querySelectorAll('a[href],button,summary')].filter(n=>n.getBoundingClientRect().width>0&&n.getBoundingClientRect().height>0);
      const i=controls.indexOf(document.activeElement),next=i+(e.shiftKey?-1:1);
      if(i>=0){e.preventDefault();e.stopImmediatePropagation();if(next<0||next>=controls.length){toggle.click();toggle.focus()}else controls[next].focus()}
    }
  },true);
  document.addEventListener('keyup',e=>{
    if(e.key==='Escape'&&escapeKeyUpPending){
      escapeKeyUpPending=false;
      e.preventDefault();e.stopPropagation();
    }
  },true);
  document.addEventListener('click',e=>{if(details.open&&!details.contains(e.target))details.open=false});
})();
