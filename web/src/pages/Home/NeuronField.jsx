import React, { useEffect, useRef } from 'react';

const PARTICLE_COLOR = '0, 243, 255';
const PARTICLE_RADIUS_MIN = 0.5;
const PARTICLE_RADIUS_MAX = 2.5;
const REPULSE_RADIUS = 150;
const MOUSE_LINK_RADIUS = 225;
const PARTICLE_LINK_RADIUS = 118;

function randomInRange(min, max) {
  return Math.random() * (max - min) + min;
}

function createParticle(width, height) {
  const baseX = Math.random() * width;
  const baseY = Math.random() * height;

  return {
    x: baseX,
    y: baseY,
    baseX,
    baseY,
    baseVX: randomInRange(-0.16, 0.16),
    baseVY: randomInRange(-0.14, 0.14),
    radius: randomInRange(PARTICLE_RADIUS_MIN, PARTICLE_RADIUS_MAX),
    alpha: randomInRange(0.35, 1),
  };
}

function createParticles(width, height) {
  const density = Math.max(
    48,
    Math.min(112, Math.floor((width * height) / 18000)),
  );
  return Array.from({ length: density }, () => createParticle(width, height));
}

export default function NeuronField() {
  const canvasRef = useRef(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) {
      return undefined;
    }

    const context = canvas.getContext('2d');
    if (!context) {
      return undefined;
    }

    let animationFrameId = 0;
    let width = 0;
    let height = 0;
    let devicePixelRatio = 1;
    let particles = [];

    const mouse = {
      x: 0,
      y: 0,
      active: false,
    };

    const resizeCanvas = () => {
      width = window.innerWidth;
      height = window.innerHeight;
      devicePixelRatio = Math.min(window.devicePixelRatio || 1, 2);

      canvas.width = Math.floor(width * devicePixelRatio);
      canvas.height = Math.floor(height * devicePixelRatio);
      canvas.style.width = `${width}px`;
      canvas.style.height = `${height}px`;

      context.setTransform(devicePixelRatio, 0, 0, devicePixelRatio, 0, 0);
      particles = createParticles(width, height);
    };

    const updateMouse = (clientX, clientY) => {
      mouse.x = clientX;
      mouse.y = clientY;
      mouse.active = true;
    };

    const handleMouseMove = (event) => {
      updateMouse(event.clientX, event.clientY);
    };

    const handleTouchMove = (event) => {
      const touch = event.touches[0];
      if (!touch) {
        return;
      }
      updateMouse(touch.clientX, touch.clientY);
    };

    const handleMouseLeave = () => {
      mouse.active = false;
    };

    const drawParticle = (particle) => {
      context.beginPath();
      context.fillStyle = `rgba(${PARTICLE_COLOR}, ${particle.alpha})`;
      context.shadowBlur = 14;
      context.shadowColor = `rgba(${PARTICLE_COLOR}, 0.72)`;
      context.arc(particle.x, particle.y, particle.radius, 0, Math.PI * 2);
      context.fill();
    };

    const drawMouseCore = () => {
      if (!mouse.active) {
        return;
      }

      context.beginPath();
      context.fillStyle = `rgba(${PARTICLE_COLOR}, 0.95)`;
      context.shadowBlur = 28;
      context.shadowColor = `rgba(${PARTICLE_COLOR}, 0.9)`;
      context.arc(mouse.x, mouse.y, 2.2, 0, Math.PI * 2);
      context.fill();

      context.beginPath();
      context.lineWidth = 1;
      context.strokeStyle = `rgba(${PARTICLE_COLOR}, 0.16)`;
      context.arc(mouse.x, mouse.y, REPULSE_RADIUS, 0, Math.PI * 2);
      context.stroke();
    };

    const updateParticle = (particle) => {
      particle.baseX += particle.baseVX;
      particle.baseY += particle.baseVY;

      if (particle.baseX <= 0 || particle.baseX >= width) {
        particle.baseVX *= -1;
        particle.baseX = Math.max(0, Math.min(width, particle.baseX));
      }

      if (particle.baseY <= 0 || particle.baseY >= height) {
        particle.baseVY *= -1;
        particle.baseY = Math.max(0, Math.min(height, particle.baseY));
      }

      if (mouse.active) {
        const deltaX = particle.x - mouse.x;
        const deltaY = particle.y - mouse.y;
        const distance = Math.hypot(deltaX, deltaY) || 0.001;

        if (distance < REPULSE_RADIUS) {
          const force = (REPULSE_RADIUS - distance) / REPULSE_RADIUS;
          const push = force * 11;
          particle.x += (deltaX / distance) * push;
          particle.y += (deltaY / distance) * push;
          return;
        }
      }

      particle.x += (particle.baseX - particle.x) / 50;
      particle.y += (particle.baseY - particle.y) / 50;
    };

    const animate = () => {
      context.clearRect(0, 0, width, height);

      for (let index = 0; index < particles.length; index += 1) {
        updateParticle(particles[index]);
      }

      context.shadowBlur = 0;

      for (let index = 0; index < particles.length; index += 1) {
        const particle = particles[index];

        for (
          let nextIndex = index + 1;
          nextIndex < particles.length;
          nextIndex += 1
        ) {
          const nextParticle = particles[nextIndex];
          const deltaX = particle.x - nextParticle.x;
          const deltaY = particle.y - nextParticle.y;
          const distance = Math.hypot(deltaX, deltaY);

          if (distance >= PARTICLE_LINK_RADIUS) {
            continue;
          }

          const opacity =
            ((PARTICLE_LINK_RADIUS - distance) / PARTICLE_LINK_RADIUS) * 0.26;
          context.beginPath();
          context.lineWidth = 0.5;
          context.strokeStyle = `rgba(${PARTICLE_COLOR}, ${opacity})`;
          context.moveTo(particle.x, particle.y);
          context.lineTo(nextParticle.x, nextParticle.y);
          context.stroke();
        }

        if (mouse.active) {
          const deltaX = particle.x - mouse.x;
          const deltaY = particle.y - mouse.y;
          const distance = Math.hypot(deltaX, deltaY);

          if (distance < MOUSE_LINK_RADIUS) {
            const opacity =
              ((MOUSE_LINK_RADIUS - distance) / MOUSE_LINK_RADIUS) * 0.38;
            context.beginPath();
            context.lineWidth = 1;
            context.strokeStyle = `rgba(${PARTICLE_COLOR}, ${opacity})`;
            context.moveTo(particle.x, particle.y);
            context.lineTo(mouse.x, mouse.y);
            context.stroke();
          }
        }
      }

      particles.forEach(drawParticle);
      drawMouseCore();
      context.shadowBlur = 0;

      animationFrameId = window.requestAnimationFrame(animate);
    };

    resizeCanvas();
    animate();

    window.addEventListener('resize', resizeCanvas);
    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseleave', handleMouseLeave);
    window.addEventListener('touchmove', handleTouchMove, { passive: true });
    window.addEventListener('touchend', handleMouseLeave);

    return () => {
      window.cancelAnimationFrame(animationFrameId);
      window.removeEventListener('resize', resizeCanvas);
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseleave', handleMouseLeave);
      window.removeEventListener('touchmove', handleTouchMove);
      window.removeEventListener('touchend', handleMouseLeave);
    };
  }, []);

  return (
    <div className='home-landing__neuron-layer' aria-hidden='true'>
      <canvas ref={canvasRef} className='home-landing__neuron-canvas' />
    </div>
  );
}
